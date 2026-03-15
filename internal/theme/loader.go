package theme

import (
	"fmt"
	"os"

	"charm.land/lipgloss/v2"
	"gopkg.in/yaml.v3"
)

type ThemeConfig struct {
	Colors struct {
		Title   string `yaml:"title"`
		Success string `yaml:"success"`
		Warning string `yaml:"warning"`
		Error   string `yaml:"error"`
		Info    string `yaml:"info"`
		Subtle  string `yaml:"subtle"`
		Path    string `yaml:"path"`
		Branch  string `yaml:"branch"`
		Remote  string `yaml:"remote"`
		Status  string `yaml:"status"`
		Stats   string `yaml:"stats"`
	} `yaml:"colors"`
	Styles struct {
		TitleBold bool `yaml:"title_bold"`
		PathBold  bool `yaml:"path_bold"`
	} `yaml:"styles"`
}

func LoadThemeFromFile(path string) (Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultTheme, fmt.Errorf("failed to read theme file: %w", err)
	}

	var config ThemeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return DefaultTheme, fmt.Errorf("failed to parse theme file: %w", err)
	}

	theme := Theme{
		Title:   createStyle(config.Colors.Title, config.Styles.TitleBold),
		Success: createStyle(config.Colors.Success, false),
		Warning: createStyle(config.Colors.Warning, false),
		Error:   createStyle(config.Colors.Error, false),
		Info:    createStyle(config.Colors.Info, false),
		Subtle:  createStyle(config.Colors.Subtle, false),
		Path:    createStyle(config.Colors.Path, config.Styles.PathBold),
		Branch:  createStyle(config.Colors.Branch, false),
		Remote:  createStyle(config.Colors.Remote, false),
		Status:  createStyle(config.Colors.Status, false),
		Stats:   createStyle(config.Colors.Stats, false),
	}

	return theme, nil
}

func createStyle(color string, bold bool) lipgloss.Style {
	style := lipgloss.NewStyle()

	if color != "" {
		style = style.Foreground(lipgloss.Color(color))
	}

	if bold {
		style = style.Bold(true)
	}

	return style
}

func LoadTheme(themePath string) Theme {
	if themePath == "" {
		return DefaultTheme
	}

	theme, err := LoadThemeFromFile(themePath)
	if err != nil {
		return DefaultTheme
	}

	return theme
}

func ExportDefaultTheme(path string) error {
	config := ThemeConfig{}
	config.Colors.Title = "63"
	config.Colors.Success = "46"
	config.Colors.Warning = "214"
	config.Colors.Error = "196"
	config.Colors.Info = "39"
	config.Colors.Subtle = "241"
	config.Colors.Path = "33"
	config.Colors.Branch = "141"
	config.Colors.Remote = "220"
	config.Colors.Status = "46"
	config.Colors.Stats = "250"
	config.Styles.TitleBold = true
	config.Styles.PathBold = true

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal theme: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write theme file: %w", err)
	}

	return nil
}
