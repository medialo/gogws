package gws2

type WorkspaceFormat interface {
	Name() string
	Detect(files ConfigFiles) (WorkspaceConfig, bool)
	New(dir string) WorkspaceConfig
}

type WorkspaceConfig interface {
	Format() WorkspaceFormat
	Path() string
	Read() (*Workspace, error)
	WriteWorkspaces(w *Workspace) error
	WriteProjects(w *Workspace) error
	ClearWorkspaces() (string, error)
	ClearProjects() (string, error)
}
