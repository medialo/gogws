package gws2

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/medialo/gogws/internal/git"
)

type ConfigFile struct {
	Path   string
	Legacy bool
}

type Repository interface {
	Id() int
	GetPath() string
	GetName() string
	IsGitRepository() bool
	FolderExist() bool
	// Index returns the shared path index for the tree this Repository
	// belongs to, or nil if it was never attached to one (e.g. built
	// outside of Loader.Load()).
	Index() *Index
}

// Entry is the common identity of anything registered under a workspace: a
// resolved absolute path, the path as written in its config file, a
// display name, and whether it exists on disk. It makes no claim about
// git — see GitRepository for that — and no claim about project-vs-
// workspace-ness, which is simply the concrete Go type (Project or
// Workspace) that embeds it.
type Entry struct {
	id           int
	AbsolutePath string // Absolute filesystem path (rootPath joined with RelativePath); use GetPath() to read it
	RelativePath string // Path exactly as written in the gws config file (e.g. "sub/project" or "."), unmodified; written back as-is by formatBaseRepository so save never leaks a machine-specific absolute path
	Name         string // Name represents the display name, based on the last part of the path
	FolderExists bool
	index        *Index
}

func (e *Entry) Id() int {
	return e.id
}

func (e *Entry) GetPath() string {
	return e.AbsolutePath
}

func (e *Entry) GetName() string {
	return e.Name
}

func (e *Entry) FolderExist() bool {
	return e.FolderExists
}

func (e *Entry) Index() *Index {
	return e.index
}

// setIndex attaches the shared path index. Unexported: only the loader and
// AddProject/AddWorkspace are expected to wire this up.
func (e *Entry) setIndex(idx *Index) {
	e.index = idx
}

// effectivePath returns RelativePath if set, else AbsolutePath — used when
// writing an entry back to its .gws config file so a save never leaks a
// machine-specific absolute path.
func (e *Entry) effectivePath() string {
	if e.RelativePath != "" {
		return e.RelativePath
	}
	return e.AbsolutePath
}

// GitRepository is an Entry that is always git-backed: a Project always
// is. (A sub-Workspace may or may not be — see Workspace.)
type GitRepository struct {
	Entry
	Remotes []*git.Remote
}

func (gr *GitRepository) IsGitRepository() bool {
	return len(gr.Remotes) > 0
}

func (gr *GitRepository) GetOriginRemote() *git.Remote {
	return originRemote(gr.Remotes)
}

func (gr *GitRepository) GetUpstreamRemote() *git.Remote {
	return upstreamRemote(gr.Remotes)
}

func (gr *GitRepository) formatBaseRepository() string {
	return formatEntryLine(gr.effectivePath(), gr.Remotes)
}

type Project struct {
	GitRepository
}

type Workspace struct {
	Entry
	Remotes             []*git.Remote // nil for a plain folder workspace, populated for a git-backed sub-workspace
	Error               error
	Projects            []*Project
	Children            []*Workspace
	WorkspaceConfigFile *ConfigFile
	ProjectConfigFile   *ConfigFile
}

func (w *Workspace) IsGitRepository() bool {
	return len(w.Remotes) > 0
}

func (w *Workspace) GetOriginRemote() *git.Remote {
	return originRemote(w.Remotes)
}

func (w *Workspace) GetUpstreamRemote() *git.Remote {
	return upstreamRemote(w.Remotes)
}

func (w *Workspace) formatBaseRepository() string {
	return formatEntryLine(w.effectivePath(), w.Remotes)
}

func originRemote(remotes []*git.Remote) *git.Remote {
	if len(remotes) > 0 {
		return remotes[0]
	}
	return nil
}

func upstreamRemote(remotes []*git.Remote) *git.Remote {
	if len(remotes) > 1 {
		return remotes[1]
	}
	return nil
}

// formatEntryLine renders one .gws config line for path given its remotes.
// A folder entry (no remotes) round-trips via the literal "folder" marker
// that parseWorkspaceLine recognizes.
func formatEntryLine(path string, remotes []*git.Remote) string {
	if len(remotes) == 0 {
		return fmt.Sprintf("%s | folder", path)
	}
	var remoteParts []string
	for _, remote := range remotes {
		if remote.Name == "origin" {
			remoteParts = append(remoteParts, remote.URL)
		} else {
			remoteParts = append(remoteParts, fmt.Sprintf("%s %s", remote.URL, remote.Name))
		}
	}
	return fmt.Sprintf("%s | %s", path, strings.Join(remoteParts, " | "))
}

func NewRootWorkspace(path string) *Workspace {
	return &Workspace{
		Entry: Entry{
			id:           -1,
			AbsolutePath: path,
			FolderExists: true,
			Name:         filepath.Base(path),
		},
		Projects: []*Project{},
		Children: []*Workspace{},
	}
}

// NewProject builds a new git-backed Project rooted at rootPath, for a
// repository at relativePath exactly as it should appear in
// .projects.gws. It is not attached to any workspace or index yet; call
// Workspace.AddProject to attach it, then Workspace.SaveProjects to persist.
func NewProject(rootPath, relativePath string, remotes []*git.Remote) *Project {
	absPath := filepath.Join(rootPath, relativePath)
	_, err := os.Stat(absPath)
	return &Project{
		GitRepository: GitRepository{
			Entry: Entry{
				AbsolutePath: absPath,
				RelativePath: relativePath,
				Name:         filepath.Base(absPath),
				FolderExists: err == nil,
			},
			Remotes: remotes,
		},
	}
}

// NewChildWorkspace builds a new git-backed Workspace rooted at rootPath,
// for a repository at relativePath exactly as it should appear in
// .workspaces.gws. It is not attached to any parent workspace or index
// yet; call Workspace.AddWorkspace to attach it, then
// Workspace.SaveWorkspace to persist.
func NewChildWorkspace(rootPath, relativePath string, remote *git.Remote) *Workspace {
	absPath := filepath.Join(rootPath, relativePath)
	_, err := os.Stat(absPath)
	return &Workspace{
		Entry: Entry{
			AbsolutePath: absPath,
			RelativePath: relativePath,
			Name:         filepath.Base(absPath),
			FolderExists: err == nil,
		},
		Remotes:  []*git.Remote{remote},
		Projects: []*Project{},
		Children: []*Workspace{},
	}
}

// NewFolderWorkspace builds a new plain-folder Workspace (no git remote,
// written back with the literal "folder" marker) rooted at rootPath, for
// relativePath exactly as it should appear in .workspaces.gws.
func NewFolderWorkspace(rootPath, relativePath string) *Workspace {
	absPath := filepath.Join(rootPath, relativePath)
	_, err := os.Stat(absPath)
	return &Workspace{
		Entry: Entry{
			AbsolutePath: absPath,
			RelativePath: relativePath,
			Name:         filepath.Base(absPath),
			FolderExists: err == nil,
		},
		Projects: []*Project{},
		Children: []*Workspace{},
	}
}

// todo: this is a hack, remove letter
func (w *Workspace) AddSelfWorkspace() {
	ws := &Workspace{
		Entry: Entry{
			id:           -1,
			AbsolutePath: "p",
			FolderExists: false,
			Name:         "n",
		},
		Remotes: w.Remotes,
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

// RemoveProject removes the project at path from this workspace or any
// descendant workspace, and rebuilds the shared index. Returns the
// workspace the project belonged to (so its own .projects.gws can be
// re-saved), or nil if no matching project was found anywhere in the tree.
func (w *Workspace) RemoveProject(path string) *Workspace {
	owner := w.removeProjectAnywhere(path)
	if owner != nil {
		w.ReindexAll()
	}
	return owner
}

func (w *Workspace) removeProjectAnywhere(path string) *Workspace {
	key := pathKey(path)
	for i, p := range w.Projects {
		if pathKey(p.AbsolutePath) == key {
			w.Projects = append(w.Projects[:i], w.Projects[i+1:]...)
			return w
		}
	}
	for _, c := range w.Children {
		if owner := c.removeProjectAnywhere(path); owner != nil {
			return owner
		}
	}
	return nil
}

// RemoveWorkspace removes the child workspace at path from this workspace
// or any descendant workspace, and rebuilds the shared index. Returns the
// workspace the child belonged to (so its own .workspaces.gws can be
// re-saved), or nil if no matching workspace was found anywhere in the
// tree.
func (w *Workspace) RemoveWorkspace(path string) *Workspace {
	owner := w.removeWorkspaceAnywhere(path)
	if owner != nil {
		w.ReindexAll()
	}
	return owner
}

func (w *Workspace) removeWorkspaceAnywhere(path string) *Workspace {
	key := pathKey(path)
	for i, c := range w.Children {
		if pathKey(c.AbsolutePath) == key {
			w.Children = append(w.Children[:i], w.Children[i+1:]...)
			return w
		}
	}
	for _, c := range w.Children {
		if owner := c.removeWorkspaceAnywhere(path); owner != nil {
			return owner
		}
	}
	return nil
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
	_, workspaceConfigLocation := getWorkspacesConfigFileLocation(w.AbsolutePath)

	if err := os.MkdirAll(filepath.Dir(workspaceConfigLocation.Path), 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", ConfigDirName, err)
	}

	file, err := os.OpenFile(workspaceConfigLocation.Path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open workspaces file: %w", err)
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
	_, projectConfigLocation := getProjectsConfigFileLocation(w.AbsolutePath)

	if err := os.MkdirAll(filepath.Dir(projectConfigLocation.Path), 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", ConfigDirName, err)
	}

	file, err := os.OpenFile(projectConfigLocation.Path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
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

func (w *Workspace) MissingWorkspaces() []*Workspace {
	var missings = make([]*Workspace, 0, len(w.Children))
	for _, child := range w.Children {
		_, err := os.Stat(child.AbsolutePath)
		if os.IsNotExist(err) {
			missings = append(missings, child)
		}
	}
	return missings
}

func (w *Workspace) MissingProjects() []*Project {
	var missings = make([]*Project, 0, len(w.Projects))
	for _, project := range w.Projects {
		_, err := os.Stat(project.AbsolutePath)
		if os.IsNotExist(err) {
			missings = append(missings, project)
		}
	}
	return missings
}

// MissingWorkspacesRecursive returns MissingWorkspaces for w plus every
// descendant workspace already present on disk. A descendant that is
// itself missing contributes nothing below it yet, since nothing was
// parsed from its config files; it is picked up once it exists on disk and
// the tree is reloaded.
func (w *Workspace) MissingWorkspacesRecursive() []*Workspace {
	missing := w.MissingWorkspaces()
	for _, child := range w.Children {
		missing = append(missing, child.MissingWorkspacesRecursive()...)
	}
	return missing
}

// MissingProjectsRecursive returns MissingProjects for w plus every
// descendant workspace already present on disk.
func (w *Workspace) MissingProjectsRecursive() []*Project {
	missing := w.MissingProjects()
	for _, child := range w.Children {
		missing = append(missing, child.MissingProjectsRecursive()...)
	}
	return missing
}
