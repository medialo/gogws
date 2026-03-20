package gws2

import (
	"fmt"
	"os"
	"strings"
)

type ConfigFile struct {
	Path   string
	Legacy bool
}

type Remote struct {
	Name string
	URL  string
}

//type IBAseRepository interface {
//	Path() string
//	Remotes() []*Remote
//	Exists() bool
//	gitRepository() bool
//}

type Repository interface {
	GetPath() string
}

func (br *BaseRepository) GetPath() string {
	return br.Path
}

type BaseRepository struct {
	Path    string
	Remotes []*Remote
	Exists  bool
}

type Project struct {
	BaseRepository
}

type Workspace struct {
	BaseRepository
	Root                string
	Name                string
	Error               error
	Projects            []*Project
	Children            []*Workspace
	WorkspaceConfigFile *ConfigFile
	ProjectConfigFile   *ConfigFile
}

func (w *Workspace) AddWorkspace(childWorkspace *Workspace) {
	w.Children = append(w.Children, childWorkspace)
}

func (w *Workspace) AddProject(project *Project) {
	w.Projects = append(w.Projects, project)
}

func (w *Workspace) SaveAll() error {
	err := w.SaveWorkspace()
	if err != nil {
		return err
	}
	err = w.SaveProjects()
	if err != nil {
		return err
	}
	return nil
}

func (w *Workspace) SaveWorkspace() error {
	_, workspaceConfigLocation := getWorkspacesConfigFileLocation(w.Path)

	file, err := os.OpenFile(workspaceConfigLocation.Path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open projects file: %w", err)
	}
	defer file.Close()

	lines := make([]string, 0, len(w.Children))

	for _, ws := range w.Children {
		lines = append(lines, ws.formatBaseRepository())
	}

	_, err = file.WriteString(strings.Join(lines, "\n"))
	if err != nil {
		return err
	}

	return nil
}

func (w *Workspace) SaveProjects() error {
	_, projectConfigLocation := getProjectsConfigFileLocation(w.Path)

	file, err := os.OpenFile(projectConfigLocation.Path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open projects file: %w", err)
	}
	defer file.Close()

	lines := make([]string, 0, len(w.Projects))

	for _, project := range w.Projects {
		lines = append(lines, project.formatBaseRepository())
	}

	_, err = file.WriteString(strings.Join(lines, "\n"))
	if err != nil {
		return err
	}

	return nil
}

func (br *BaseRepository) formatBaseRepository() string {
	var remoteParts []string
	for _, remote := range br.Remotes {
		if remote.Name == "origin" {
			remoteParts = append(remoteParts, remote.URL)
		} else {
			remoteParts = append(remoteParts, fmt.Sprintf("%s %s", remote.URL, remote.Name))
		}
	}
	return fmt.Sprintf("%s | %s", br.Path, strings.Join(remoteParts, " | "))
}

func (br *BaseRepository) isGitRepository() bool {
	return len(br.Remotes) > 0
}
