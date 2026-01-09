package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ResultsTable displays query results in a table format
type ResultsTable struct {
	columns     []string
	rows        [][]string
	columnTypes []string
	cursor      int
	offset      int
	height      int
	width       int
	styles      *Styles
	focused     bool
	scrollX     int
	maxColWidth int
}

// NewResultsTable creates a new results table
func NewResultsTable() ResultsTable {
	return ResultsTable{
		columns:     []string{},
		rows:        [][]string{},
		columnTypes: []string{},
		cursor:      0,
		offset:      0,
		height:      20,
		width:       120,
		styles:      DefaultStyles(),
		focused:     false,
		scrollX:     0,
		maxColWidth: 40,
	}
}

// SetData sets the table data
func (t *ResultsTable) SetData(columns []string, columnTypes []string, rows [][]string) {
	t.columns = columns
	t.columnTypes = columnTypes
	t.rows = rows
	t.cursor = 0
	t.offset = 0
	t.scrollX = 0
}

// Clear clears the table data
func (t *ResultsTable) Clear() {
	t.columns = []string{}
	t.columnTypes = []string{}
	t.rows = [][]string{}
	t.cursor = 0
	t.offset = 0
	t.scrollX = 0
}

// SetSize sets the table dimensions
func (t *ResultsTable) SetSize(width, height int) {
	t.width = width
	t.height = height
}

// Focus focuses the table
func (t *ResultsTable) Focus() {
	t.focused = true
}

// Blur removes focus from the table
func (t *ResultsTable) Blur() {
	t.focused = false
}

// IsFocused returns whether the table is focused
func (t ResultsTable) IsFocused() bool {
	return t.focused
}

// RowCount returns the number of rows
func (t ResultsTable) RowCount() int {
	return len(t.rows)
}

// Update handles messages
func (t ResultsTable) Update(msg tea.Msg) (ResultsTable, tea.Cmd) {
	if !t.focused {
		return t, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if t.cursor > 0 {
				t.cursor--
				if t.cursor < t.offset {
					t.offset = t.cursor
				}
			}
		case "down", "j":
			if t.cursor < len(t.rows)-1 {
				t.cursor++
				if t.cursor >= t.offset+t.visibleRows() {
					t.offset = t.cursor - t.visibleRows() + 1
				}
			}
		case "left", "h":
			if t.scrollX > 0 {
				t.scrollX--
			}
		case "right", "l":
			if t.scrollX < len(t.columns)-1 {
				t.scrollX++
			}
		case "pgup":
			t.cursor -= t.visibleRows()
			if t.cursor < 0 {
				t.cursor = 0
			}
			t.offset = t.cursor
		case "pgdown":
			t.cursor += t.visibleRows()
			if t.cursor >= len(t.rows) {
				t.cursor = len(t.rows) - 1
			}
			if t.cursor >= t.offset+t.visibleRows() {
				t.offset = t.cursor - t.visibleRows() + 1
			}
		case "home", "g":
			t.cursor = 0
			t.offset = 0
		case "end", "G":
			t.cursor = len(t.rows) - 1
			if t.cursor >= t.visibleRows() {
				t.offset = t.cursor - t.visibleRows() + 1
			}
		}
	}

	return t, nil
}

func (t ResultsTable) visibleRows() int {
	return t.height - 4 // Account for header and borders
}

// View renders the table
func (t ResultsTable) View() string {
	if len(t.columns) == 0 {
		return t.styles.Muted.Render("No results to display")
	}

	var b strings.Builder

	// Calculate column widths
	colWidths := t.calculateColumnWidths()

	// Header
	headerCells := make([]string, 0)
	for i := t.scrollX; i < len(t.columns) && i < t.scrollX+t.visibleColumns(colWidths); i++ {
		cell := truncateString(t.columns[i], colWidths[i])
		cell = padRight(cell, colWidths[i])
		headerCells = append(headerCells, t.styles.Bold.Foreground(ColorSecondary).Render(cell))
	}
	header := strings.Join(headerCells, " | ")

	borderStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	if t.focused {
		borderStyle = borderStyle.Foreground(ColorPrimary)
	}

	b.WriteString(borderStyle.Render("┌" + strings.Repeat("─", len(stripAnsi(header))+2) + "┐"))
	b.WriteString("\n")
	b.WriteString(borderStyle.Render("│ ") + header + borderStyle.Render(" │"))
	b.WriteString("\n")
	b.WriteString(borderStyle.Render("├" + strings.Repeat("─", len(stripAnsi(header))+2) + "┤"))
	b.WriteString("\n")

	// Rows
	visibleEnd := t.offset + t.visibleRows()
	if visibleEnd > len(t.rows) {
		visibleEnd = len(t.rows)
	}

	for i := t.offset; i < visibleEnd; i++ {
		row := t.rows[i]
		rowCells := make([]string, 0)

		for j := t.scrollX; j < len(row) && j < t.scrollX+t.visibleColumns(colWidths); j++ {
			cell := truncateString(row[j], colWidths[j])
			cell = padRight(cell, colWidths[j])

			// Style based on type and selection
			if i == t.cursor && t.focused {
				cell = t.styles.Selected.Render(cell)
			} else if i%2 == 0 {
				cell = t.styles.TableRow.Render(cell)
			} else {
				cell = t.styles.TableRowAlt.Render(cell)
			}
			rowCells = append(rowCells, cell)
		}

		rowStr := strings.Join(rowCells, " | ")
		b.WriteString(borderStyle.Render("│ ") + rowStr + borderStyle.Render(" │"))
		b.WriteString("\n")
	}

	b.WriteString(borderStyle.Render("└" + strings.Repeat("─", len(stripAnsi(header))+2) + "┘"))
	b.WriteString("\n")

	// Footer with info
	info := fmt.Sprintf("Row %d/%d | Column %d/%d",
		t.cursor+1, len(t.rows),
		t.scrollX+1, len(t.columns))
	b.WriteString(t.styles.Muted.Render(info))

	return b.String()
}

func (t ResultsTable) calculateColumnWidths() []int {
	if len(t.columns) == 0 {
		return nil
	}

	widths := make([]int, len(t.columns))

	// Start with column header widths
	for i, col := range t.columns {
		widths[i] = len(col)
	}

	// Check row widths
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Cap at max width
	for i := range widths {
		if widths[i] > t.maxColWidth {
			widths[i] = t.maxColWidth
		}
	}

	return widths
}

func (t ResultsTable) visibleColumns(colWidths []int) int {
	available := t.width - 4 // Borders
	count := 0
	used := 0

	for i := t.scrollX; i < len(colWidths); i++ {
		needed := colWidths[i] + 3 // Column + separator
		if used+needed > available {
			break
		}
		used += needed
		count++
	}

	if count == 0 {
		count = 1
	}
	return count
}

// GetSelectedRow returns the currently selected row
func (t ResultsTable) GetSelectedRow() []string {
	if t.cursor >= 0 && t.cursor < len(t.rows) {
		return t.rows[t.cursor]
	}
	return nil
}

// Helper functions

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}

// stripAnsi removes ANSI escape codes from a string
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false

	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}

	return result.String()
}
