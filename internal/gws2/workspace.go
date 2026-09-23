package gws2

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/medialo/gogws/internal/git"
)

type RepositoryType int

//go:generate go tool enumer -type=RepositoryType -trimprefix RepositoryType
const (
	RepositoryTypeProject RepositoryType = iota
	RepositoryTypeWorkspace
	RepositoryTypeFolder
)

type ConfigFile struct {
	Path   string
	Legacy bool
}

type Repository interface {
	Id() int
	GetPath() string
	GetName() string
	GetType() RepositoryType
	IsGitRepository() bool
	FolderExist() bool
	// Index returns the shared path index for the tree this Repository
	// belongs to, or nil if it was never attached to one (e.g. built
	// outside of Loader.Load()).
	Index() *Index
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

func (gr *GitRepository) FolderExist() bool {
	return gr.FolderExists
}

func (gr *GitRepository) Index() *Index {
	return gr.index
}

// setIndex attaches the shared path index. Unexported: only the loader and
// AddProject/AddWorkspace are expected to wire this up.
func (gr *GitRepository) setIndex(idx *Index) {
	gr.index = idx
}

type GitRepository struct {
	id            int
	Path          string // Path of the repository from the gws config file, can be ".", use GetPath() to get real path
	Name          string // Name represents the name of the Git repository based on the last part of the path
	Remotes       []*git.Remote
	FolderExists  bool
	gitRepository bool
	Type          RepositoryType
	index         *Index
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

func NewRootWorkspace(path string) *Workspace {
	return &Workspace{
		id:            -1,
		Path:          path,
		FolderExists:  true,
		Name:          filepath.Base(path),
		gitRepository: git.IsGitFolder(path),
		Projects:      []*Project{},
		Children:      []*Workspace{},
	}
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

func (w *Workspace) FlattenRepositories() []Repository {
	repos := make([]Repository, 0, len(w.Projects)+len(w.Children))
	for _, project := range w.Projects {
		repos = append(repos, project)
	}
	for _, childWorkspace := range w.Children {
		repos = append(repos, childWorkspace)
		repos = append(repos, childWorkspace.FlattenRepositories()...)
	}
	return repos
}

func (w *Workspace) AddWorkspace(childWorkspace *Workspace) {
	w.Children = append(w.Children, childWorkspace)
	if idx := w.Index(); idx != nil {
		childWorkspace.setIndex(idx)
		idx.put(childWorkspace)
	}
}

func (w *Workspace) AddProject(project *Project) {
	w.Projects = append(w.Projects, project)
	if idx := w.Index(); idx != nil {
		project.setIndex(idx)
		idx.put(project)
	}
}

func (w *Workspace) ReindexAll() {
	if w.Index() == nil {
		w.setIndex(NewIndex())
	}
	w.Index().Rebuild(w)
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

func (gr *GitRepository) GetOriginRemote() *git.Remote {
	if len(gr.Remotes) > 0 {
		return gr.Remotes[0]
	}
	return nil
}

func (gr *GitRepository) GetUpstreamRemote() *git.Remote {
	if len(gr.Remotes) > 1 {
		return gr.Remotes[1]
	}
	return nil
}

func (w *Workspace) MissingWorkspaces() []*Workspace {
	var missings = make([]*Workspace, 0, len(w.Children))
	for _, child := range w.Children {
		//if child.Type == RepositoryTypeFolder {
		//	continue
		//}
		_, err := os.Stat(child.Path)
		if os.IsNotExist(err) {
			missings = append(missings, child)
		}
	}
	return missings
}

func (w *Workspace) MissingProjects() []*Project {
	var missings = make([]*Project, 0, len(w.Projects))
	for _, project := range w.Projects {
		_, err := os.Stat(project.Path)
		if os.IsNotExist(err) {
			missings = append(missings, project)
		}
	}
	return missings
}
