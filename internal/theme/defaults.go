package theme

import (
	"charm.land/lipgloss/v2"
)

const (
	marginBase = 2
)

type Theme struct {
	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	SubtitleI lipgloss.Style
	Success   lipgloss.Style
	Warning   lipgloss.Style
	Error     lipgloss.Style
	Info      lipgloss.Style
	Subtle    lipgloss.Style
	Path      lipgloss.Style
	Branch    lipgloss.Style
	Remote    lipgloss.Style
	Status    lipgloss.Style
	Stats     lipgloss.Style
	Ahead     lipgloss.Style
	Behind    lipgloss.Style

	MarginLeft     lipgloss.Style
	MarginLeftFunc func(int) lipgloss.Style

	HeaderBox  lipgloss.Style
	SummaryBox lipgloss.Style

	Icons Icons
	Emoji Emoji
}

type Icons struct {
	Success   string
	Warning   string
	Error     string
	Info      string
	Pending   string
	Workspace string
	Root      string
	Folder    string
}

type Emoji struct {
	Folder string
}

var DefaultTheme = Theme{
	Title:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")),
	Subtitle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245")),
	SubtitleI: lipgloss.NewStyle().Bold(true).Italic(true).Foreground(lipgloss.Color("245")),
	Success:   lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
	Warning:   lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
	Error:     lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
	Info:      lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
	Subtle:    lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	Path:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75")),
	Branch:    lipgloss.NewStyle().Foreground(lipgloss.Color("213")),
	Remote:    lipgloss.NewStyle().Foreground(lipgloss.Color("220")),
	Status:    lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
	Stats:     lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
	Ahead:     lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
	Behind:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")),

	MarginLeft: lipgloss.NewStyle().MarginLeft(marginBase),
	MarginLeftFunc: func(i int) lipgloss.Style {
		return lipgloss.NewStyle().MarginLeft(i * marginBase)
	},

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
		Root:      "⁜",
		Folder: lipgloss.NewStyle().
			Foreground(lipgloss.Color("227")).
			Bold(true).Render("🗀"),
	},

	Emoji: Emoji{
		Folder: "📁",
	},
}

var currentTheme = DefaultTheme

func GetTheme() Theme {
	return currentTheme
}

func SetTheme(t Theme) {
	currentTheme = t
}
