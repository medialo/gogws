package loader

import (
	"sync"

	"github.com/medialo/gogws/internal/gws2"
)

type lazyConfig struct {
	dir    string
	once   sync.Once
	config gws2.WorkspaceConfig
}

func newLazyConfig(dir string) *lazyConfig {
	return &lazyConfig{dir: dir}
}

func (l *lazyConfig) resolve() gws2.WorkspaceConfig {
	l.once.Do(func() {
		l.config = ConfigFor(l.dir)
	})
	return l.config
}

func (l *lazyConfig) Format() gws2.WorkspaceFormat {
	return l.resolve().Format()
}

func (l *lazyConfig) Path() string {
	return l.resolve().Path()
}

func (l *lazyConfig) Read() (*gws2.Workspace, error) {
	return l.resolve().Read()
}

func (l *lazyConfig) WriteWorkspaces(w *gws2.Workspace) error {
	return l.resolve().WriteWorkspaces(w)
}

func (l *lazyConfig) WriteProjects(w *gws2.Workspace) error {
	return l.resolve().WriteProjects(w)
}

func (l *lazyConfig) ClearWorkspaces() (string, error) {
	return l.resolve().ClearWorkspaces()
}

func (l *lazyConfig) ClearProjects() (string, error) {
	return l.resolve().ClearProjects()
}
