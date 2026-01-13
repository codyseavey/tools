package ui

import (
	"sort"
	"strings"
	"unicode"

	"github.com/codyseavey/tools/azlogs/internal/azure"
)

// ContextType represents what kind of completion is expected
type ContextType int

const (
	ContextUnknown ContextType = iota
	ContextTableName            // Start of query, after union/join
	ContextOperator             // After | pipe
	ContextColumnName           // After where, project, by, etc.
	ContextFunction             // In summarize, expecting aggregation
	ContextValue                // After ==, in string literal (no suggestions)
)

// ParsedContext contains information about the cursor position and what's expected
type ParsedContext struct {
	Type             ContextType
	CurrentTable     string   // The main table being queried
	CurrentWord      string   // Partial word at cursor
	WordStartPos     int      // Position where current word starts
	ReferencedTables []string // All tables referenced in query
	AfterKeyword     string   // The keyword before current position (e.g., "where", "project")
}

// Suggestion represents an autocomplete suggestion
type Suggestion struct {
	Text        string // The completion text
	Type        string // "table", "column", "keyword", "function", "operator"
	Description string // Additional info (e.g., column type)
	Score       int    // Relevance score for sorting
}

// KQL operators that appear after |
var kqlOperators = []string{
	"where", "project", "extend", "summarize", "join", "union",
	"take", "limit", "top", "sort", "order", "distinct",
	"count", "render", "parse", "evaluate", "invoke",
	"mv-expand", "make-series", "serialize", "range",
}

// KQL aggregation functions for summarize
var kqlFunctions = []string{
	"count()", "sum(", "avg(", "min(", "max(", "dcount(",
	"percentile(", "stdev(", "variance(", "countif(",
	"sumif(", "avgif(", "minif(", "maxif(",
	"make_list(", "make_set(", "arg_max(", "arg_min(",
}

// KQL comparison operators
var kqlComparisons = []string{
	"==", "!=", "<", ">", "<=", ">=",
	"contains", "has", "startswith", "endswith",
	"matches regex", "in", "!in", "between",
	"and", "or", "not",
}

// Time functions
var kqlTimeFunctions = []string{
	"ago(", "now()", "datetime(", "timespan(",
	"startofday(", "startofweek(", "startofmonth(",
	"endofday(", "endofweek(", "endofmonth(",
	"bin(", "format_datetime(",
}

// AutocompleteEngine provides instant local autocomplete suggestions
type AutocompleteEngine struct {
	tables  []string
	schemas map[string][]azure.Column
}

// NewAutocompleteEngine creates a new autocomplete engine
func NewAutocompleteEngine() *AutocompleteEngine {
	return &AutocompleteEngine{
		schemas: make(map[string][]azure.Column),
	}
}

// SetTables updates the available tables
func (e *AutocompleteEngine) SetTables(tables []string) {
	e.tables = tables
}

// SetSchemas updates the schema cache
func (e *AutocompleteEngine) SetSchemas(schemas map[string][]azure.Column) {
	e.schemas = schemas
}

// ParseContext analyzes the query at cursor position to determine context
func (e *AutocompleteEngine) ParseContext(query string, cursorPos int) ParsedContext {
	ctx := ParsedContext{
		Type:         ContextUnknown,
		WordStartPos: cursorPos,
	}

	if cursorPos > len(query) {
		cursorPos = len(query)
	}

	// Get text before cursor
	beforeCursor := query[:cursorPos]
	beforeCursor = strings.TrimRight(beforeCursor, " \t\n")

	// Find current word being typed
	ctx.CurrentWord, ctx.WordStartPos = e.findCurrentWord(beforeCursor)

	// Find referenced tables
	ctx.ReferencedTables = e.findReferencedTables(query)
	if len(ctx.ReferencedTables) > 0 {
		ctx.CurrentTable = ctx.ReferencedTables[0]
	}

	// Determine context type
	ctx.Type, ctx.AfterKeyword = e.determineContextType(beforeCursor)

	return ctx
}

// findCurrentWord extracts the word being typed at cursor
func (e *AutocompleteEngine) findCurrentWord(text string) (string, int) {
	if len(text) == 0 {
		return "", 0
	}

	// Find word boundary going backwards
	end := len(text)
	start := end

	for start > 0 {
		r := rune(text[start-1])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			break
		}
		start--
	}

	return text[start:end], start
}

// findReferencedTables extracts table names from the query
func (e *AutocompleteEngine) findReferencedTables(query string) []string {
	var tables []string
	seen := make(map[string]bool)
	queryLower := strings.ToLower(query)

	for _, table := range e.tables {
		tableLower := strings.ToLower(table)

		// Check if table appears at start or after pipe/union/join
		patterns := []string{
			tableLower + " ",
			tableLower + "|",
			tableLower + "\n",
			"| " + tableLower,
			"|" + tableLower,
			"union " + tableLower,
			"join " + tableLower,
			"join (" + tableLower,
		}

		for _, pattern := range patterns {
			if strings.Contains(queryLower, pattern) || strings.HasPrefix(queryLower, tableLower) {
				if !seen[table] {
					tables = append(tables, table)
					seen[table] = true
				}
				break
			}
		}
	}

	return tables
}

// determineContextType figures out what kind of suggestions to show
func (e *AutocompleteEngine) determineContextType(beforeCursor string) (ContextType, string) {
	trimmed := strings.TrimSpace(beforeCursor)

	// Empty or just starting - suggest tables
	if len(trimmed) == 0 {
		return ContextTableName, ""
	}

	// Check if we're right after a pipe
	if strings.HasSuffix(trimmed, "|") {
		return ContextOperator, ""
	}

	// Check if last non-word char is pipe with space
	lastPipe := strings.LastIndex(trimmed, "|")
	if lastPipe != -1 {
		afterPipe := strings.TrimSpace(trimmed[lastPipe+1:])
		afterPipeLower := strings.ToLower(afterPipe)

		// Just after pipe, might be typing operator
		if len(afterPipe) == 0 || !strings.Contains(afterPipe, " ") {
			return ContextOperator, ""
		}

		// Check for keywords that expect columns
		columnKeywords := []string{"where ", "project ", "extend ", "by ", "on "}
		for _, kw := range columnKeywords {
			if strings.Contains(afterPipeLower, kw) {
				// Find the keyword
				idx := strings.LastIndex(afterPipeLower, kw)
				if idx != -1 {
					afterKw := afterPipe[idx+len(kw):]
					// If there's content after keyword and no operator yet
					if len(strings.TrimSpace(afterKw)) >= 0 {
						return ContextColumnName, strings.TrimSpace(kw)
					}
				}
			}
		}

		// Check for summarize - expect functions
		if strings.Contains(afterPipeLower, "summarize ") {
			return ContextFunction, "summarize"
		}

		// Check for join/union - expect tables
		if strings.HasSuffix(afterPipeLower, "join ") || strings.HasSuffix(afterPipeLower, "union ") {
			return ContextTableName, ""
		}
	}

	// At the very start, suggest tables
	if !strings.Contains(trimmed, "|") && !strings.Contains(trimmed, " ") {
		return ContextTableName, ""
	}

	// Default to unknown
	return ContextUnknown, ""
}

// GetSuggestions returns suggestions based on context
func (e *AutocompleteEngine) GetSuggestions(ctx ParsedContext, limit int) []Suggestion {
	var suggestions []Suggestion

	switch ctx.Type {
	case ContextTableName:
		suggestions = e.getTableSuggestions(ctx.CurrentWord)
	case ContextOperator:
		suggestions = e.getOperatorSuggestions(ctx.CurrentWord)
	case ContextColumnName:
		suggestions = e.getColumnSuggestions(ctx.CurrentTable, ctx.CurrentWord)
	case ContextFunction:
		suggestions = e.getFunctionSuggestions(ctx.CurrentWord)
	default:
		// Mixed suggestions
		suggestions = append(suggestions, e.getOperatorSuggestions(ctx.CurrentWord)...)
		suggestions = append(suggestions, e.getColumnSuggestions(ctx.CurrentTable, ctx.CurrentWord)...)
	}

	// Sort by score descending
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Score > suggestions[j].Score
	})

	// Limit results
	if limit > 0 && len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	return suggestions
}

func (e *AutocompleteEngine) getTableSuggestions(prefix string) []Suggestion {
	var suggestions []Suggestion
	prefixLower := strings.ToLower(prefix)

	for _, table := range e.tables {
		tableLower := strings.ToLower(table)
		if strings.HasPrefix(tableLower, prefixLower) {
			score := 100
			if tableLower == prefixLower {
				score = 200 // Exact match
			}
			suggestions = append(suggestions, Suggestion{
				Text:        table,
				Type:        "table",
				Description: "Table",
				Score:       score,
			})
		}
	}

	return suggestions
}

func (e *AutocompleteEngine) getOperatorSuggestions(prefix string) []Suggestion {
	var suggestions []Suggestion
	prefixLower := strings.ToLower(prefix)

	for _, op := range kqlOperators {
		if strings.HasPrefix(op, prefixLower) {
			score := 100
			if op == prefixLower {
				score = 200
			}
			// Boost common operators
			if op == "where" || op == "project" || op == "take" || op == "summarize" {
				score += 50
			}
			suggestions = append(suggestions, Suggestion{
				Text:        op,
				Type:        "operator",
				Description: "Operator",
				Score:       score,
			})
		}
	}

	return suggestions
}

func (e *AutocompleteEngine) getColumnSuggestions(tableName, prefix string) []Suggestion {
	var suggestions []Suggestion
	prefixLower := strings.ToLower(prefix)

	columns, ok := e.schemas[tableName]
	if !ok {
		return suggestions
	}

	for _, col := range columns {
		colLower := strings.ToLower(col.Name)
		if strings.HasPrefix(colLower, prefixLower) {
			score := 100
			if colLower == prefixLower {
				score = 200
			}
			// Boost common columns
			if col.Name == "TimeGenerated" || col.Name == "ResourceId" || col.Name == "OperationName" {
				score += 50
			}
			suggestions = append(suggestions, Suggestion{
				Text:        col.Name,
				Type:        "column",
				Description: col.Type,
				Score:       score,
			})
		}
	}

	return suggestions
}

func (e *AutocompleteEngine) getFunctionSuggestions(prefix string) []Suggestion {
	var suggestions []Suggestion
	prefixLower := strings.ToLower(prefix)

	allFunctions := append(kqlFunctions, kqlTimeFunctions...)

	for _, fn := range allFunctions {
		fnLower := strings.ToLower(fn)
		if strings.HasPrefix(fnLower, prefixLower) {
			score := 100
			// Boost common functions
			if strings.HasPrefix(fnLower, "count") || strings.HasPrefix(fnLower, "sum") {
				score += 50
			}
			suggestions = append(suggestions, Suggestion{
				Text:        fn,
				Type:        "function",
				Description: "Function",
				Score:       score,
			})
		}
	}

	return suggestions
}
