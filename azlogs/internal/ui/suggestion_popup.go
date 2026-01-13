package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SuggestionPopup displays autocomplete suggestions
type SuggestionPopup struct {
	suggestions   []Suggestion
	selectedIndex int
	visible       bool
	maxVisible    int
	scrollOffset  int
	width         int
	styles        *PopupStyles
}

// PopupStyles defines styling for the suggestion popup
type PopupStyles struct {
	Box           lipgloss.Style
	Item          lipgloss.Style
	SelectedItem  lipgloss.Style
	TypeIcon      lipgloss.Style
	Description   lipgloss.Style
}

// NewSuggestionPopup creates a new suggestion popup
func NewSuggestionPopup() *SuggestionPopup {
	return &SuggestionPopup{
		maxVisible: 8,
		width:      50,
		styles:     DefaultPopupStyles(),
	}
}

// DefaultPopupStyles returns default popup styling
func DefaultPopupStyles() *PopupStyles {
	return &PopupStyles{
		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1),
		Item: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		SelectedItem: lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("255")).
			Bold(true),
		TypeIcon: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		Description: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true),
	}
}

// SetSuggestions updates the suggestions list
func (p *SuggestionPopup) SetSuggestions(suggestions []Suggestion) {
	p.suggestions = suggestions
	p.selectedIndex = 0
	p.scrollOffset = 0
	p.visible = len(suggestions) > 0
}

// Show makes the popup visible
func (p *SuggestionPopup) Show() {
	p.visible = true
}

// Hide hides the popup
func (p *SuggestionPopup) Hide() {
	p.visible = false
	p.suggestions = nil
	p.selectedIndex = 0
	p.scrollOffset = 0
}

// IsVisible returns whether the popup is visible
func (p *SuggestionPopup) IsVisible() bool {
	return p.visible && len(p.suggestions) > 0
}

// Next moves selection down
func (p *SuggestionPopup) Next() {
	if len(p.suggestions) == 0 {
		return
	}
	p.selectedIndex++
	if p.selectedIndex >= len(p.suggestions) {
		p.selectedIndex = 0
		p.scrollOffset = 0
	}
	// Adjust scroll
	if p.selectedIndex >= p.scrollOffset+p.maxVisible {
		p.scrollOffset = p.selectedIndex - p.maxVisible + 1
	}
}

// Previous moves selection up
func (p *SuggestionPopup) Previous() {
	if len(p.suggestions) == 0 {
		return
	}
	p.selectedIndex--
	if p.selectedIndex < 0 {
		p.selectedIndex = len(p.suggestions) - 1
		p.scrollOffset = max(0, len(p.suggestions)-p.maxVisible)
	}
	// Adjust scroll
	if p.selectedIndex < p.scrollOffset {
		p.scrollOffset = p.selectedIndex
	}
}

// Selected returns the currently selected suggestion
func (p *SuggestionPopup) Selected() *Suggestion {
	if len(p.suggestions) == 0 || p.selectedIndex >= len(p.suggestions) {
		return nil
	}
	return &p.suggestions[p.selectedIndex]
}

// GetSelectedText returns the text of the selected suggestion
func (p *SuggestionPopup) GetSelectedText() string {
	if s := p.Selected(); s != nil {
		return s.Text
	}
	return ""
}

// SetWidth sets the popup width
func (p *SuggestionPopup) SetWidth(width int) {
	p.width = width
}

// typeIcon returns an icon for the suggestion type
func typeIcon(t string) string {
	switch t {
	case "table":
		return "◆"
	case "column":
		return "○"
	case "operator":
		return "●"
	case "function":
		return "ƒ"
	case "keyword":
		return "◇"
	default:
		return "·"
	}
}

// View renders the popup
func (p *SuggestionPopup) View() string {
	if !p.visible || len(p.suggestions) == 0 {
		return ""
	}

	var lines []string

	// Calculate visible range
	endIdx := p.scrollOffset + p.maxVisible
	if endIdx > len(p.suggestions) {
		endIdx = len(p.suggestions)
	}

	// Max width for text
	maxTextWidth := p.width - 10 // Account for icon, padding, description

	for i := p.scrollOffset; i < endIdx; i++ {
		s := p.suggestions[i]

		// Icon
		icon := p.styles.TypeIcon.Render(typeIcon(s.Type))

		// Text (truncate if needed)
		text := s.Text
		if len(text) > maxTextWidth {
			text = text[:maxTextWidth-3] + "..."
		}

		// Description
		desc := ""
		if s.Description != "" && s.Description != s.Type {
			desc = p.styles.Description.Render(" " + s.Description)
		}

		// Build line
		line := fmt.Sprintf("%s %s%s", icon, text, desc)

		// Apply selection style
		if i == p.selectedIndex {
			// Pad to width for full highlight
			padded := line
			for len(padded) < p.width-4 {
				padded += " "
			}
			line = p.styles.SelectedItem.Render(padded)
		} else {
			line = p.styles.Item.Render(line)
		}

		lines = append(lines, line)
	}

	// Add scroll indicator if needed
	if len(p.suggestions) > p.maxVisible {
		scrollInfo := fmt.Sprintf(" %d/%d ", p.selectedIndex+1, len(p.suggestions))
		lines = append(lines, p.styles.TypeIcon.Render(scrollInfo))
	}

	content := strings.Join(lines, "\n")
	return p.styles.Box.Width(p.width).Render(content)
}

// CompactView renders a single-line preview of the top suggestion
func (p *SuggestionPopup) CompactView() string {
	if !p.visible || len(p.suggestions) == 0 {
		return ""
	}

	s := p.suggestions[p.selectedIndex]
	icon := typeIcon(s.Type)
	return fmt.Sprintf("%s %s", icon, s.Text)
}
