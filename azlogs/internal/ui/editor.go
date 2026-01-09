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

// HighlightKQL applies basic syntax highlighting to KQL
func HighlightKQL(query string) string {
	result := query

	// Highlight keywords
	keywordStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	for _, kw := range kqlKeywords {
		// Case-insensitive replacement
		lowerQuery := strings.ToLower(result)
		lowerKw := strings.ToLower(kw)

		for {
			idx := strings.Index(strings.ToLower(result), lowerKw)
			if idx == -1 {
				break
			}

			// Check word boundaries
			isWordStart := idx == 0 || !isAlphaNum(result[idx-1])
			isWordEnd := idx+len(kw) == len(result) || !isAlphaNum(result[idx+len(kw)])

			if isWordStart && isWordEnd {
				highlighted := keywordStyle.Render(result[idx : idx+len(kw)])
				result = result[:idx] + highlighted + result[idx+len(kw):]
				// Move past the highlighted text
				break
			}
			_ = lowerQuery // Suppress unused variable warning
			break
		}
	}

	return result
}

func isAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}
