package standard

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gws2"
)

type Format struct{}

var _ gws2.WorkspaceFormat = Format{}

func (Format) Name() string {
	return "standard"
}

func (f Format) Detect(dir string) (gws2.WorkspaceConfig, bool) {
	location := locate(dir)
	if !location.Exists {
		return nil, false
	}
	location.warnIgnored()
	return &config{dir: dir, location: location}, true
}

func (f Format) New(dir string) gws2.WorkspaceConfig {
	return &config{dir: dir, location: locate(dir)}
}

type config struct {
	dir      string
	location fileLocation
}

func (c *config) Format() gws2.WorkspaceFormat {
	return Format{}
}

func (c *config) Path() string {
	return c.location.Path
}

func (c *config) Read() (*gws2.Workspace, error) {
	doc, err := readDocument(c.location)
	if err != nil {
		return nil, err
	}
	return toWorkspace(c.dir, doc)
}

func (c *config) WriteWorkspaces(w *gws2.Workspace) error {
	doc, err := readDocument(c.location)
	if err != nil {
		return err
	}
	doc.Self = nil
	if w.SelfEntry != nil {
		doc.Self = &selfEntry{Remotes: fromRemotes(w.SelfEntry.Remotes)}
	}
	doc.Workspaces = make([]entry, 0, len(w.Children))
	for _, child := range w.Children {
		doc.Workspaces = append(doc.Workspaces, entry{Path: child.ConfigPath(), Remotes: fromRemotes(child.Remotes)})
	}
	return c.write(doc)
}

func (c *config) WriteProjects(w *gws2.Workspace) error {
	doc, err := readDocument(c.location)
	if err != nil {
		return err
	}
	doc.Projects = make([]entry, 0, len(w.Projects))
	for _, project := range w.Projects {
		doc.Projects = append(doc.Projects, entry{Path: project.ConfigPath(), Remotes: fromRemotes(project.Remotes)})
	}
	return c.write(doc)
}

func (c *config) ClearWorkspaces() (string, error) {
	return c.clearSection(func(doc *document) {
		doc.Self = nil
		doc.Workspaces = nil
	})
}

func (c *config) ClearProjects() (string, error) {
	return c.clearSection(func(doc *document) {
		doc.Projects = nil
	})
}

func (c *config) write(doc *document) error {
	if err := writeDocument(c.location, doc); err != nil {
		return err
	}
	c.location.Exists = true
	return nil
}

func (c *config) clearSection(clear func(doc *document)) (string, error) {
	if !c.location.Exists {
		return "", nil
	}
	doc, err := readDocument(c.location)
	if err != nil {
		return c.location.Path, err
	}
	clear(doc)
	if doc.isEmpty() {
		if err := os.Remove(c.location.Path); err != nil {
			return c.location.Path, fmt.Errorf("failed to delete %s: %w", c.location.Path, err)
		}
		c.location.Exists = false
		return c.location.Path, nil
	}
	return c.location.Path, c.write(doc)
}

func readDocument(location fileLocation) (*document, error) {
	doc := &document{Version: CurrentVersion}
	if !location.Exists {
		return doc, nil
	}
	data, err := os.ReadFile(location.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", location.Path, err)
	}
	if err := location.Encoding.decode(data, doc); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", location.Path, err)
	}
	if doc.Version == 0 {
		doc.Version = CurrentVersion
	}
	if doc.Version != CurrentVersion {
		return nil, fmt.Errorf("unsupported %s version %d (supported: %d)", location.Path, doc.Version, CurrentVersion)
	}
	return doc, nil
}

func writeDocument(location fileLocation, doc *document) error {
	if err := os.MkdirAll(filepath.Dir(location.Path), 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", filepath.Dir(location.Path), err)
	}
	data, err := location.Encoding.encode(doc)
	if err != nil {
		return fmt.Errorf("failed to encode %s: %w", location.Path, err)
	}
	if err := os.WriteFile(location.Path, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", location.Path, err)
	}
	return nil
}

func toWorkspace(dir string, doc *document) (*gws2.Workspace, error) {
	node := gws2.NewRootWorkspace(dir)

	if doc.Self != nil {
		remotes, err := toRemotes(doc.Self.Remotes)
		if err != nil {
			return nil, fmt.Errorf("self: %w", err)
		}
		node.SelfEntry = newWorkspace(dir, ".", remotes)
	}

	for i, e := range doc.Workspaces {
		if e.Path == "" {
			return nil, fmt.Errorf("workspaces[%d]: empty path", i)
		}
		remotes, err := toRemotes(e.Remotes)
		if err != nil {
			return nil, fmt.Errorf("workspaces[%d] %s: %w", i, e.Path, err)
		}
		ws := newWorkspace(dir, e.Path, remotes)
		if e.Path == "." || gws2.PathKey(ws.AbsolutePath) == gws2.PathKey(dir) {
			node.SelfEntry = ws
			continue
		}
		node.AddWorkspace(ws)
	}

	for i, e := range doc.Projects {
		if e.Path == "" {
			return nil, fmt.Errorf("projects[%d]: empty path", i)
		}
		remotes, err := toRemotes(e.Remotes)
		if err != nil {
			return nil, fmt.Errorf("projects[%d] %s: %w", i, e.Path, err)
		}
		if len(remotes) == 0 {
			return nil, fmt.Errorf("projects[%d] %s: no remotes defined", i, e.Path)
		}
		node.AddProject(gws2.NewProject(dir, e.Path, remotes))
	}

	return node, nil
}

func newWorkspace(dir, path string, remotes []*git.Remote) *gws2.Workspace {
	if len(remotes) == 0 {
		return gws2.NewFolderWorkspace(dir, path)
	}
	return gws2.NewChildWorkspace(dir, path, remotes)
}

func toRemotes(remotes []remote) ([]*git.Remote, error) {
	result := make([]*git.Remote, 0, len(remotes))
	for i, r := range remotes {
		if r.URL == "" {
			return nil, fmt.Errorf("remotes[%d]: empty url", i)
		}
		name := r.Name
		if name == "" {
			name = defaultRemoteName(i)
		}
		result = append(result, &git.Remote{Name: name, URL: r.URL})
	}
	return result, nil
}

func fromRemotes(remotes []*git.Remote) []remote {
	if len(remotes) == 0 {
		return nil
	}
	result := make([]remote, 0, len(remotes))
	for _, r := range remotes {
		result = append(result, remote{Name: r.Name, URL: r.URL})
	}
	return result
}

func defaultRemoteName(index int) string {
	if index == 1 {
		return "upstream"
	}
	return "origin"
}
