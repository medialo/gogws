package gitignore

import (
	"bytes"
	"os"
	"path/filepath"
	"text/template"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/gws"
)

type TemplateData struct {
	Extension      string
	ConfigDir      string
	ProjectsFile   string
	WorkspacesFile string
}

func DefaultData() TemplateData {
	return TemplateData{
		Extension:      gws.FileExtension,
		ConfigDir:      gws.ConfigDirName,
		ProjectsFile:   gws.ProjectsFileName,
		WorkspacesFile: gws.WorkspacesFileName,
	}
}

func loadTemplate() (*template.Template, error) {
	customPath, err := getCustomTemplatePath()
	if err == nil {
		if _, statErr := os.Stat(customPath); statErr == nil {
			content, readErr := os.ReadFile(customPath)
			if readErr == nil {
				return template.New("gitignore").Parse(string(content))
			}
		}
	}

	return template.New("gitignore").Parse(DefaultTemplate)
}

func getCustomTemplatePath() (string, error) {
	configDir, err := config.GetUserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, gws.TemplatesDirName, "gitignore.tmpl"), nil
}

func hasCustomTemplate() bool {
	customPath, err := getCustomTemplatePath()
	if err != nil {
		return false
	}
	_, err = os.Stat(customPath)
	return err == nil
}

func renderTemplate(data TemplateData) (string, error) {
	tmpl, err := loadTemplate()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
