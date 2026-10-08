package gws

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gws2"
)

type Format struct{}

var _ gws2.WorkspaceFormat = Format{}

func (Format) Name() string {
	return "gws"
}

func (f Format) Detect(dir string) (gws2.WorkspaceConfig, bool) {
	c := newConfig(dir)
	if !c.projects.Exists && !c.workspaces.Exists {
		return nil, false
	}
	c.projects.warnIgnored()
	c.workspaces.warnIgnored()
	return c, true
}

func (f Format) New(dir string) gws2.WorkspaceConfig {
	return newConfig(dir)
}

type config struct {
	dir        string
	projects   fileLocation
	workspaces fileLocation
}

func newConfig(dir string) *config {
	return &config{dir: dir, projects: locateProjectsFile(dir), workspaces: locateWorkspacesFile(dir)}
}

func (c *config) Format() gws2.WorkspaceFormat {
	return Format{}
}

func (c *config) Path() string {
	if !c.projects.Exists && c.workspaces.Exists {
		return c.workspaces.Path
	}
	return c.projects.Path
}

func (c *config) Read() (*gws2.Workspace, error) {
	node := gws2.NewRootWorkspace(c.dir)
	var errs []error

	if c.projects.Exists {
		projects, err := parseProjectsFile(c.dir, c.projects.Path)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to read projects: %w", err))
		}
		for _, p := range projects {
			node.AddProject(p)
		}
	}

	if c.workspaces.Exists {
		workspaces, err := parseWorkspacesFile(c.dir, c.workspaces.Path)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to read workspaces: %w", err))
		}
		for _, ws := range workspaces {
			if ws.RelativePath == "." || gws2.PathKey(ws.AbsolutePath) == gws2.PathKey(c.dir) {
				node.SelfEntry = ws
				continue
			}
			node.AddWorkspace(ws)
		}
	}

	return node, errors.Join(errs...)
}

func (c *config) WriteWorkspaces(w *gws2.Workspace) error {
	lines := make([]string, 0, len(w.Children)+1)
	if w.SelfEntry != nil {
		lines = append(lines, formatEntryLine(w.SelfEntry.ConfigPath(), w.SelfEntry.Remotes))
	}
	for _, ws := range w.Children {
		lines = append(lines, formatEntryLine(ws.ConfigPath(), ws.Remotes))
	}
	return writeLines(&c.workspaces, lines)
}

func (c *config) WriteProjects(w *gws2.Workspace) error {
	lines := make([]string, 0, len(w.Projects))
	for _, project := range w.Projects {
		lines = append(lines, formatEntryLine(project.ConfigPath(), project.Remotes))
	}
	return writeLines(&c.projects, lines)
}

func (c *config) ClearWorkspaces() (string, error) {
	return removeFile(&c.workspaces)
}

func (c *config) ClearProjects() (string, error) {
	return removeFile(&c.projects)
}

func writeLines(location *fileLocation, lines []string) error {
	if err := os.MkdirAll(filepath.Dir(location.Path), 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", filepath.Dir(location.Path), err)
	}
	if err := os.WriteFile(location.Path, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", location.Path, err)
	}
	location.Exists = true
	return nil
}

func removeFile(location *fileLocation) (string, error) {
	if !location.Exists {
		return "", nil
	}
	if err := os.Remove(location.Path); err != nil {
		return location.Path, fmt.Errorf("failed to delete %s: %w", location.Path, err)
	}
	location.Exists = false
	return location.Path, nil
}

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
