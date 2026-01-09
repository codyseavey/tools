package ui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary = lipgloss.Color("#06B6D4") // Cyan
	ColorSuccess   = lipgloss.Color("#10B981") // Green
	ColorWarning   = lipgloss.Color("#F59E0B") // Amber
	ColorError     = lipgloss.Color("#EF4444") // Red
	ColorMuted     = lipgloss.Color("#6B7280") // Gray
	ColorBorder    = lipgloss.Color("#374151") // Dark gray
	ColorBg        = lipgloss.Color("#1F2937") // Background
	ColorBgAlt     = lipgloss.Color("#111827") // Alt background
)

// Styles contains all UI styles
type Styles struct {
	Title        lipgloss.Style
	Subtitle     lipgloss.Style
	StatusBar    lipgloss.Style
	StatusBarKey lipgloss.Style
	StatusBarVal lipgloss.Style
	Error        lipgloss.Style
	Success      lipgloss.Style
	Warning      lipgloss.Style
	Muted        lipgloss.Style
	Bold         lipgloss.Style
	Header       lipgloss.Style
	Table        lipgloss.Style
	TableHeader  lipgloss.Style
	TableRow     lipgloss.Style
	TableRowAlt  lipgloss.Style
	Selected     lipgloss.Style
	Prompt       lipgloss.Style
	Input        lipgloss.Style
	Help         lipgloss.Style
	HelpKey      lipgloss.Style
	HelpDesc     lipgloss.Style
	Box          lipgloss.Style
	ActiveBox    lipgloss.Style
	Spinner      lipgloss.Style
}

// DefaultStyles returns the default style configuration
func DefaultStyles() *Styles {
	return &Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1),

		Subtitle: lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Italic(true),

		StatusBar: lipgloss.NewStyle().
			Background(ColorBgAlt).
			Padding(0, 1),

		StatusBarKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary),

		StatusBarVal: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")),

		Error: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorError),

		Success: lipgloss.NewStyle().
			Foreground(ColorSuccess),

		Warning: lipgloss.NewStyle().
			Foreground(ColorWarning),

		Muted: lipgloss.NewStyle().
			Foreground(ColorMuted),

		Bold: lipgloss.NewStyle().
			Bold(true),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(ColorBorder).
			PaddingBottom(1).
			MarginBottom(1),

		Table: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1),

		TableHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(ColorBorder),

		TableRow: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")),

		TableRowAlt: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB")),

		Selected: lipgloss.NewStyle().
			Bold(true).
			Background(ColorPrimary).
			Foreground(lipgloss.Color("#FFFFFF")),

		Prompt: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary),

		Input: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1),

		Help: lipgloss.NewStyle().
			Foreground(ColorMuted),

		HelpKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary),

		HelpDesc: lipgloss.NewStyle().
			Foreground(ColorMuted),

		Box: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2),

		ActiveBox: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2),

		Spinner: lipgloss.NewStyle().
			Foreground(ColorPrimary),
	}
}

// Logo returns the ASCII art logo
func Logo() string {
	return `
   _   _____  _     ___   ___ ___
  /_\ |_  / | |   / _ \ / __/ __|
 / _ \ / /| |__ | (_) | (_ \__ \
/_/ \_\___|____| \___/ \___|___/

  Azure Log Analytics CLI
`
}

// LogoStyled returns a styled version of the logo
func LogoStyled() string {
	return lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Render(Logo())
}
