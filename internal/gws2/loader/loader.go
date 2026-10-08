package loader

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/gws2/format/gws"
	"github.com/medialo/gogws/internal/gws2/format/standard"
)

type Loader struct {
	root      string
	recursive bool
	maxDepth  int
	runDoctor bool
}

func NewFromAutoRoot() (*Loader, error) {
	path, err := FindRoot()

	if err != nil {
		return nil, err
	}

	return NewFromPath(path), nil
}

func NewFromPath(path string) *Loader {
	return &Loader{
		root:      path,
		recursive: true,
		maxDepth:  gws2.DefaultMaxDepth,
		runDoctor: true,
	}
}

func (l *Loader) RunDoctor(enabled bool) *Loader {
	l.runDoctor = enabled
	return l
}

func (l *Loader) Recursive(enabled bool) *Loader {
	l.recursive = enabled
	return l
}

func (l *Loader) MaxDepth(depth int) *Loader {
	l.maxDepth = depth
	return l
}

func (l *Loader) Load() (*gws2.Workspace, error) {
	slog.Debug("Loading workspace loader...", "path", l.root)
	ws, err := l.loadRecursiveInit(l.root)
	if ws == nil || (l.runDoctor && !ws.IsValid()) {
		return nil, fmt.Errorf("workspace is in invalid state, please run 'gogws doctor' to show diagnostics")
	}
	slog.Debug("Found projects and workspaces", "projects", len(ws.Projects), "workspaces", len(ws.Children), "rootIsGit", ws.IsGitRepository())
	return ws, err
}

func (l *Loader) loadRecursiveInit(rootPath string) (*gws2.Workspace, error) {
	ws := gws2.NewRootWorkspace(rootPath)
	ws.AttachIndex(gws2.NewIndex())

	ws, err := l.loadRecursive(ws, rootPath, 0)
	if ws != nil && len(ws.Remotes) == 0 && ws.SelfEntry != nil {
		ws.Remotes = ws.SelfEntry.Remotes
	}
	return ws, err
}

func (l *Loader) loadRecursive(current *gws2.Workspace, rootPath string, depth int) (*gws2.Workspace, error) {
	idx := current.Index()
	if idx.Has(rootPath) {
		slog.Debug("Skipping already visited workspace", "path", rootPath)
		current.Error = fmt.Errorf("workspace already visited (possible cycle): %s", rootPath)
		return current, nil
	}
	idx.Register(current)

	if depth > l.maxDepth {
		slog.Warn("Maximum workspace depth reached", "path", rootPath)
		current.Error = fmt.Errorf("maximum workspace depth (%d) reached at: %s", l.maxDepth, rootPath)
		return current, nil
	}

	config := ConfigFor(rootPath)
	current.BindConfig(config)
	slog.Debug("Loading workspace", "depth", depth, "path", rootPath, "format", config.Format().Name())

	node, err := config.Read()
	if err != nil {
		slog.Warn("Failed to read workspace configuration", "path", rootPath, "format", config.Format().Name(), "err", err)
	}
	if node == nil {
		return current, nil
	}

	current.Settings = node.Settings
	if node.SelfEntry != nil {
		current.SelfEntry = node.SelfEntry
	}

	projects := node.Projects
	if rules, err := LoadIgnoreRules(rootPath); err == nil {
		projects = filterIgnoredProjects(projects, rules)
	}
	for _, p := range projects {
		current.AddProject(p)
	}

	for _, child := range node.Children {
		nextRootPath := child.AbsolutePath
		if !child.FolderExists || !l.recursive {
			child.BindConfig(newLazyConfig(nextRootPath))
			current.AddWorkspace(child)
			continue
		}

		child.AttachIndex(idx)
		resolved, err := l.loadRecursive(child, nextRootPath, depth+1)
		if err != nil {
			child.Error = err
			current.AddWorkspace(child)
		} else if resolved != nil {
			current.AddWorkspace(resolved)
		}
	}

	slog.Debug(" > gws2 > Loaded workspace", "path", rootPath, "projects", len(current.Projects), "children", len(current.Children))
	return current, nil
}

var (
	cachedRootDir string
)

func FindRoot() (string, error) {
	if cachedRootDir != "" {
		return cachedRootDir, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if IsWorkspace(dir) {
			cachedRootDir = dir
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no workspace found (no %s, %s or %s file found in current or parent directories)",
				filepath.Join(gws2.ConfigDirName, standard.FileBaseName+".{yaml,yml,json,toml}"), filepath.Join(gws2.ConfigDirName, gws.ProjectsFileNameInDir), gws.ProjectsFileName)
		}
		dir = parent
	}
}
