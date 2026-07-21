package gws

const (
	FileExtension      = "gws"
	ConfigDirName      = ".gws"
	HooksDirName       = "hooks"
	TemplatesDirName   = "templates"
	ProjectsFileName   = ".projects." + FileExtension
	WorkspacesFileName = ".workspaces." + FileExtension
	IgnoreFileName     = ".ignore." + FileExtension
	DefaultParallel    = 5
	DefaultMaxDepth    = 3
)

type Remote struct {
	Name string
	URL  string
}

type Project struct {
	Path    string
	Remotes []Remote
	Exists  bool
}

type Workspace struct {
	Path     string
	Root     string
	Name     string
	Remote   Remote
	Exists   bool
	Error    error
	Projects []Project
	Children []*Workspace
}

func (w *Workspace) AllProjects() []Project {
	all := make([]Project, 0, len(w.Projects))
	all = append(all, w.Projects...)
	for _, child := range w.Children {
		all = append(all, child.AllProjects()...)
	}
	return all
}

func (w *Workspace) TotalProjectCount() int {
	count := len(w.Projects)
	for _, child := range w.Children {
		count += child.TotalProjectCount()
	}
	return count
}

func (w *Workspace) TotalWorkspaceCount() int {
	count := len(w.Children)
	for _, child := range w.Children {
		count += child.TotalWorkspaceCount()
	}
	return count
}

func (w *Workspace) MissingProjects() []Project {
	var missing []Project
	for _, p := range w.Projects {
		if !p.Exists {
			missing = append(missing, p)
		}
	}
	return missing
}

func (w *Workspace) MissingWorkspaces() []*Workspace {
	var missing []*Workspace
	for _, child := range w.Children {
		if !child.Exists {
			missing = append(missing, child)
		}
	}
	return missing
}

func (w *Workspace) AllMissingProjects() []Project {
	missing := w.MissingProjects()
	for _, child := range w.Children {
		missing = append(missing, child.AllMissingProjects()...)
	}
	return missing
}

func (w *Workspace) AllMissingWorkspaces() []*Workspace {
	missing := w.MissingWorkspaces()
	for _, child := range w.Children {
		missing = append(missing, child.AllMissingWorkspaces()...)
	}
	return missing
}

func (w *Workspace) ToProject() Project {
	return Project{
		Path:    w.Path,
		Remotes: []Remote{w.Remote},
		Exists:  w.Exists,
	}
}
