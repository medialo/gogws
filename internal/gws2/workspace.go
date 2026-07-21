package gws2

import (
	"fmt"
	"os"
	"strings"
)

type RepositoryType int

//go:generate enumer -type=RepositoryType -trimprefix RepositoryType
const (
	RepositoryTypeProject RepositoryType = iota
	RepositoryTypeWorkspace
)

type ConfigFile struct {
	Path   string
	Legacy bool
}

type Remote struct {
	Name string
	URL  string
}

type Repository interface {
	Id() int
	GetPath() string
	GetName() string
	GetType() RepositoryType
	IsGitRepository() bool
}

func (gr *GitRepository) Id() int {
	return gr.id
}

func (gr *GitRepository) GetPath() string {
	return gr.Path
}

func (gr *GitRepository) GetName() string {
	return gr.Name
}

func (gr *GitRepository) GetType() RepositoryType {
	return gr.Type
}

func (gr *GitRepository) IsGitRepository() bool {
	return gr.gitRepository
}

type GitRepository struct {
	id            int
	Path          string // Path of the repository from the gws config file, can be ".", use GetPath() to get real path
	Name          string // Name represents the name of the Git repository based on the last part of the path
	Remotes       []*Remote
	FolderExists  bool
	gitRepository bool
	Type          RepositoryType
}

type Project struct {
	GitRepository
}

type Workspace struct {
	GitRepository
	Error               error
	Projects            []*Project
	Children            []*Workspace
	WorkspaceConfigFile *ConfigFile
	ProjectConfigFile   *ConfigFile
}

// todo: this is a hack, remove letter
func (w *Workspace) AddSelfWorkspace() {
	ws := &Workspace{
		GitRepository: GitRepository{
			id:           -1,
			Path:         "p",
			Remotes:      w.GitRepository.Remotes,
			FolderExists: false,
			Name:         "n",
		},
	}
	w.AddWorkspace(ws)
}

func (w *Workspace) FlattenProjects() []*Project {
	var projects []*Project
	for _, project := range w.Projects {
		projects = append(projects, project)
	}
	for _, childWorkspace := range w.Children {
		projects = append(projects, childWorkspace.FlattenProjects()...)
	}
	return projects
}

func (w *Workspace) FlattenWorkspaces() []*Workspace {
	var workspaces []*Workspace
	for _, childWorkspace := range w.Children {
		workspaces = append(workspaces, childWorkspace)
		workspaces = append(workspaces, childWorkspace.FlattenWorkspaces()...)
	}
	return workspaces
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

func (gr *GitRepository) formatBaseRepository() string {
	var remoteParts []string
	for _, remote := range gr.Remotes {
		if remote.Name == "origin" {
			remoteParts = append(remoteParts, remote.URL)
		} else {
			remoteParts = append(remoteParts, fmt.Sprintf("%s %s", remote.URL, remote.Name))
		}
	}
	return fmt.Sprintf("%s | %s", gr.Path, strings.Join(remoteParts, " | "))
}

func (gr *GitRepository) isGitRepository() bool {
	return len(gr.Remotes) > 0
}
