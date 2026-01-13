package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// KQL keywords for highlighting
var kqlKeywords = []string{
	"where", "project", "extend", "summarize", "join", "union",
	"take", "limit", "top", "sort", "order", "by", "asc", "desc",
	"count", "sum", "avg", "min", "max", "dcount", "percentile",
	"ago", "now", "datetime", "timespan", "bin", "startofday",
	"contains", "has", "startswith", "endswith", "matches",
	"and", "or", "not", "in", "between",
	"let", "set", "alias", "declare", "pattern",
	"render", "as", "on", "kind", "hint", "with",
	"parse", "extract", "split", "strcat", "tostring", "toint", "tolong", "todouble",
	"isempty", "isnotempty", "isnull", "isnotnull",
	"getschema", "search", "find", "mv-expand", "make-series",
}

// QueryEditor is a KQL query editor component
type QueryEditor struct {
	textarea    textarea.Model
	styles      *Styles
	focused     bool
	placeholder string
}

// NewQueryEditor creates a new query editor
func NewQueryEditor() QueryEditor {
	ta := textarea.New()
	ta.Placeholder = "Enter KQL query (e.g., AzureActivity | take 10)"
	ta.ShowLineNumbers = true
	ta.SetHeight(10)
	ta.SetWidth(80)
	ta.CharLimit = 0 // No limit
	ta.Focus()

	return QueryEditor{
		textarea:    ta,
		styles:      DefaultStyles(),
		focused:     true,
		placeholder: "Enter KQL query...",
	}
}

// Init initializes the editor
func (e QueryEditor) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages
func (e QueryEditor) Update(msg tea.Msg) (QueryEditor, tea.Cmd) {
	var cmd tea.Cmd
	e.textarea, cmd = e.textarea.Update(msg)
	return e, cmd
}

// View renders the editor
func (e QueryEditor) View() string {
	var b strings.Builder

	// Title
	title := e.styles.Prompt.Render("Query Editor")
	b.WriteString(title)
	b.WriteString("\n")

	// Editor
	editorStyle := e.styles.Box
	if e.focused {
		editorStyle = e.styles.ActiveBox
	}
	b.WriteString(editorStyle.Render(e.textarea.View()))

	return b.String()
}

// Focus focuses the editor
func (e *QueryEditor) Focus() {
	e.focused = true
	e.textarea.Focus()
}

// Blur removes focus from the editor
func (e *QueryEditor) Blur() {
	e.focused = false
	e.textarea.Blur()
}

// Value returns the current query text
func (e QueryEditor) Value() string {
	return e.textarea.Value()
}

// SetValue sets the query text
func (e *QueryEditor) SetValue(s string) {
	e.textarea.SetValue(s)
}

// SetSize sets the editor dimensions
func (e *QueryEditor) SetSize(width, height int) {
	e.textarea.SetWidth(width - 4) // Account for border
	e.textarea.SetHeight(height)
}

// Reset clears the editor
func (e *QueryEditor) Reset() {
	e.textarea.Reset()
}

// IsFocused returns whether the editor is focused
func (e QueryEditor) IsFocused() bool {
	return e.focused
}

// CursorPosition returns the absolute cursor position in the text
func (e QueryEditor) CursorPosition() int {
	value := e.textarea.Value()
	lines := strings.Split(value, "\n")
	line := e.textarea.Line()
	info := e.textarea.LineInfo()

	pos := 0
	for i := 0; i < line && i < len(lines); i++ {
		pos += len(lines[i]) + 1 // +1 for newline
	}
	return pos + info.CharOffset
}

// InsertText inserts text at the current cursor position
func (e *QueryEditor) InsertText(text string) {
	e.textarea.InsertString(text)
}

// Highlight styles
var (
	keywordStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#5c6bc0")).Bold(true)
	operatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff9800"))
	pipeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#757575")).Bold(true)
	stringStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#4caf50"))
	numberStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#e91e63"))
	functionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffc107"))
)

// HighlightKQL applies syntax highlighting to KQL
func HighlightKQL(query string) string {
	if query == "" {
		return query
	}

	var result strings.Builder
	i := 0

	for i < len(query) {
		// Check for pipe
		if query[i] == '|' {
			result.WriteString(pipeStyle.Render("|"))
			i++
			continue
		}

		// Check for string literals
		if query[i] == '"' || query[i] == '\'' {
			quote := query[i]
			start := i
			i++
			for i < len(query) && query[i] != quote {
				if query[i] == '\\' && i+1 < len(query) {
					i++ // Skip escaped char
				}
				i++
			}
			if i < len(query) {
				i++ // Include closing quote
			}
			result.WriteString(stringStyle.Render(query[start:i]))
			continue
		}

		// Check for numbers
		if isDigit(query[i]) {
			start := i
			for i < len(query) && (isDigit(query[i]) || query[i] == '.') {
				i++
			}
			result.WriteString(numberStyle.Render(query[start:i]))
			continue
		}

		// Check for words (keywords, identifiers)
		if isAlphaStart(query[i]) {
			start := i
			for i < len(query) && isAlphaNum(query[i]) {
				i++
			}
			word := query[start:i]

			// Check if followed by ( for function highlighting
			isFunc := i < len(query) && query[i] == '('

			// Check if it's a keyword
			if isKQLKeyword(word) {
				result.WriteString(keywordStyle.Render(word))
			} else if isFunc {
				result.WriteString(functionStyle.Render(word))
			} else {
				result.WriteString(word)
			}
			continue
		}

		// Check for comparison operators
		if i+1 < len(query) {
			twoChar := query[i : i+2]
			if twoChar == "==" || twoChar == "!=" || twoChar == "<=" || twoChar == ">=" {
				result.WriteString(operatorStyle.Render(twoChar))
				i += 2
				continue
			}
		}

		if query[i] == '<' || query[i] == '>' || query[i] == '=' {
			result.WriteString(operatorStyle.Render(string(query[i])))
			i++
			continue
		}

		// Default: write character as-is
		result.WriteByte(query[i])
		i++
	}

	return result.String()
}

func isAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

func isAlphaStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isKQLKeyword(word string) bool {
	lower := strings.ToLower(word)
	for _, kw := range kqlKeywords {
		if strings.ToLower(kw) == lower {
			return true
		}
	}
	return false
}
