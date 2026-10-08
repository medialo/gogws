package loader

import (
	"log/slog"
	"sync"

	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/gws2/format/gws"
	"github.com/medialo/gogws/internal/gws2/format/standard"
)

var formats = []gws2.WorkspaceFormat{standard.Format{}, gws.Format{}}

type detection struct {
	config gws2.WorkspaceConfig
	found  bool
}

var (
	detectedMu sync.Mutex
	detected   = map[string]detection{}
)

func Detect(dir string) (gws2.WorkspaceConfig, bool) {
	key := gws2.PathKey(dir)

	detectedMu.Lock()
	defer detectedMu.Unlock()

	if d, ok := detected[key]; ok {
		return d.config, d.found
	}

	d := detectFormats(dir)
	if d.found {
		detected[key] = d
	}
	return d.config, d.found
}

func detectFormats(dir string) detection {
	var found gws2.WorkspaceConfig
	for _, format := range formats {
		config, ok := format.Detect(dir)
		if !ok {
			continue
		}
		if found == nil {
			found = config
			continue
		}
		slog.Warn("Several workspace configurations found, using the highest priority one", "path", dir, "used", found.Path(), "ignored", config.Path())
	}
	if found == nil {
		return detection{config: formats[0].New(dir), found: false}
	}
	return detection{config: found, found: true}
}

func ConfigFor(dir string) gws2.WorkspaceConfig {
	config, _ := Detect(dir)
	return config
}

func IsWorkspace(dir string) bool {
	_, found := Detect(dir)
	return found
}

func NewRootWorkspace(dir string) *gws2.Workspace {
	ws := gws2.NewRootWorkspace(dir)
	ws.BindConfig(ConfigFor(dir))
	return ws
}

func ClearProjects(dir string) (string, error) {
	return ConfigFor(dir).ClearProjects()
}

func ClearWorkspaces(dir string) (string, error) {
	return ConfigFor(dir).ClearWorkspaces()
}
