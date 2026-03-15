package styles

import "charm.land/lipgloss/v2"

type Styles struct {
	HeaderBox  lipgloss.Style
	ContentBox lipgloss.Style
	SummaryBox lipgloss.Style

	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Success  lipgloss.Style
	Warning  lipgloss.Style
	Error    lipgloss.Style
	Info     lipgloss.Style
	Muted    lipgloss.Style

	Path   lipgloss.Style
	Branch lipgloss.Style
	Ahead  lipgloss.Style
	Behind lipgloss.Style
	Remote lipgloss.Style
	Stats  lipgloss.Style

	ConfigKey    lipgloss.Style
	ConfigValue  lipgloss.Style
	ConfigSource lipgloss.Style

	IconSuccess   string
	IconWarning   string
	IconError     string
	IconInfo      string
	IconPending   string
	IconWorkspace string
}

func DefaultStyles() *Styles {
	return &Styles{
		HeaderBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1).
			Bold(true),

		ContentBox: lipgloss.NewStyle().
			Padding(0, 2),

		SummaryBox: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1),

		Title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")),
		Subtitle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245")).MarginTop(1),
		Success:  lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		Warning:  lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		Error:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		Info:     lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
		Muted:    lipgloss.NewStyle().Foreground(lipgloss.Color("245")),

		Path:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75")),
		Branch: lipgloss.NewStyle().Foreground(lipgloss.Color("213")),
		Ahead:  lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		Behind: lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		Remote: lipgloss.NewStyle().Foreground(lipgloss.Color("220")),
		Stats:  lipgloss.NewStyle().Foreground(lipgloss.Color("250")),

		ConfigKey:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75")),
		ConfigValue:  lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		ConfigSource: lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true),

		IconSuccess:   "✓",
		IconWarning:   "●",
		IconError:     "✗",
		IconInfo:      "ℹ",
		IconPending:   "○",
		IconWorkspace: "◈",
	}
}

type StylesConfig struct {
	Colors struct {
		Header   string `yaml:"header"`
		Title    string `yaml:"title"`
		Subtitle string `yaml:"subtitle"`
		Success  string `yaml:"success"`
		Warning  string `yaml:"warning"`
		Error    string `yaml:"error"`
		Info     string `yaml:"info"`
		Muted    string `yaml:"muted"`
		Path     string `yaml:"path"`
		Branch   string `yaml:"branch"`
		Ahead    string `yaml:"ahead"`
		Behind   string `yaml:"behind"`
		Remote   string `yaml:"remote"`
		Stats    string `yaml:"stats"`
	} `yaml:"colors"`
	Icons struct {
		Success   string `yaml:"success"`
		Warning   string `yaml:"warning"`
		Error     string `yaml:"error"`
		Info      string `yaml:"info"`
		Pending   string `yaml:"pending"`
		Workspace string `yaml:"workspace"`
	} `yaml:"icons"`
}

func FromConfig(cfg StylesConfig) *Styles {
	s := DefaultStyles()

	if cfg.Colors.Header != "" {
		s.HeaderBox = s.HeaderBox.BorderForeground(lipgloss.Color(cfg.Colors.Header))
	}
	if cfg.Colors.Title != "" {
		s.Title = s.Title.Foreground(lipgloss.Color(cfg.Colors.Title))
	}
	if cfg.Colors.Success != "" {
		s.Success = s.Success.Foreground(lipgloss.Color(cfg.Colors.Success))
	}
	if cfg.Colors.Warning != "" {
		s.Warning = s.Warning.Foreground(lipgloss.Color(cfg.Colors.Warning))
	}
	if cfg.Colors.Error != "" {
		s.Error = s.Error.Foreground(lipgloss.Color(cfg.Colors.Error))
	}
	if cfg.Colors.Info != "" {
		s.Info = s.Info.Foreground(lipgloss.Color(cfg.Colors.Info))
	}
	if cfg.Colors.Muted != "" {
		s.Muted = s.Muted.Foreground(lipgloss.Color(cfg.Colors.Muted))
	}
	if cfg.Colors.Path != "" {
		s.Path = s.Path.Foreground(lipgloss.Color(cfg.Colors.Path))
	}
	if cfg.Colors.Branch != "" {
		s.Branch = s.Branch.Foreground(lipgloss.Color(cfg.Colors.Branch))
	}
	if cfg.Colors.Ahead != "" {
		s.Ahead = s.Ahead.Foreground(lipgloss.Color(cfg.Colors.Ahead))
	}
	if cfg.Colors.Behind != "" {
		s.Behind = s.Behind.Foreground(lipgloss.Color(cfg.Colors.Behind))
	}

	if cfg.Icons.Success != "" {
		s.IconSuccess = cfg.Icons.Success
	}
	if cfg.Icons.Warning != "" {
		s.IconWarning = cfg.Icons.Warning
	}
	if cfg.Icons.Error != "" {
		s.IconError = cfg.Icons.Error
	}
	if cfg.Icons.Info != "" {
		s.IconInfo = cfg.Icons.Info
	}
	if cfg.Icons.Pending != "" {
		s.IconPending = cfg.Icons.Pending
	}
	if cfg.Icons.Workspace != "" {
		s.IconWorkspace = cfg.Icons.Workspace
	}

	return s
}

var globalStyles *Styles

func init() {
	globalStyles = DefaultStyles()
}

func Get() *Styles {
	return globalStyles
}

func Set(s *Styles) {
	if s != nil {
		globalStyles = s
	}
}
