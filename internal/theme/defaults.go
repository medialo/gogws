package theme

import "charm.land/lipgloss/v2"

type Theme struct {
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Success  lipgloss.Style
	Warning  lipgloss.Style
	Error    lipgloss.Style
	Info     lipgloss.Style
	Subtle   lipgloss.Style
	Path     lipgloss.Style
	Branch   lipgloss.Style
	Remote   lipgloss.Style
	Status   lipgloss.Style
	Stats    lipgloss.Style
	Ahead    lipgloss.Style
	Behind   lipgloss.Style

	HeaderBox  lipgloss.Style
	SummaryBox lipgloss.Style

	Icons Icons
}

type Icons struct {
	Success   string
	Warning   string
	Error     string
	Info      string
	Pending   string
	Workspace string
}

var DefaultTheme = Theme{
	Title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")),
	Subtitle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245")),
	Success:  lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
	Warning:  lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
	Error:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
	Info:     lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
	Subtle:   lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	Path:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75")),
	Branch:   lipgloss.NewStyle().Foreground(lipgloss.Color("213")),
	Remote:   lipgloss.NewStyle().Foreground(lipgloss.Color("220")),
	Status:   lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
	Stats:    lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
	Ahead:    lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
	Behind:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")),

	HeaderBox: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1).
		Bold(true),

	SummaryBox: lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1),

	Icons: Icons{
		Success:   "✓",
		Warning:   "●",
		Error:     "✗",
		Info:      "ℹ",
		Pending:   "○",
		Workspace: "◈",
	},
}

var currentTheme = DefaultTheme

func GetTheme() Theme {
	return currentTheme
}

func SetTheme(t Theme) {
	currentTheme = t
}
