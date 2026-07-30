package gws2

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/medialo/gogws/internal/git"
)

type Loader struct {
	root      string
	recursive bool
	maxDepth  int
	runDoctor bool
	visited   map[string]bool
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
		maxDepth:  DefaultMaxDepth,
		runDoctor: true,
		visited:   make(map[string]bool),
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

func (l *Loader) Load() (*Workspace, error) {
	slog.Debug("Loading workspace loader...", "path", l.root)
	ws, err := l.loadRecursive(l.root, 0)
	if ws == nil || (l.runDoctor && !ws.IsValid()) {
		return nil, fmt.Errorf("workspace is in invalid state, please run 'gogws doctor' to show diagnostics")
	}
	slog.Debug("Found projects and workspaces", "projects", len(ws.Projects), "workspaces", len(ws.Children), "rootIsGit", ws.isGitRepository())
	return ws, err
}

func (l *Loader) loadRecursive(root string, depth int) (*Workspace, error) {

	actualRoot, err := os.Getwd()
	err = os.Chdir(root)
	if err != nil {
		return nil, err
	}
	defer func(dir string) {
		err := os.Chdir(dir)
		if err != nil {

		}
	}(actualRoot)

	if l.visited[root] {
		slog.Debug("Skipping already visited workspace", "path", root)
		return nil, nil
	}
	l.visited[root] = true

	if depth > l.maxDepth {
		slog.Warn("Maximum workspace depth reached", "path", root)
		return nil, nil
	}

	slog.Debug("Loading workspace", "depth", depth, "path", root)

	ws := &Workspace{
		GitRepository: GitRepository{
			id:            -1,
			Path:          root,
			FolderExists:  true,
			Name:          filepath.Base(root),
			gitRepository: git.IsGitFolder(root),
		},
		Projects: []*Project{},
		Children: []*Workspace{},
	}

	_, projectsLocation := getProjectsConfigFileLocation(root)
	if projectsLocation != nil {
		if projectsLocation.HasDuplicate {
			legacyPath := filepath.Join(root, ProjectsFileName)
			slog.Warn("Duplicate projects file found - using .gws/projects.gws, please remove the legacy file",
				"legacy", legacyPath,
				"used", projectsLocation.Path)
		}

		projects, err := parseProjectsFile(root)
		if err != nil {
			slog.Warn("Failed to read projects", "path", root, "err", err)
		} else {
			for _, p := range projects {
				projectPath := filepath.Join(root, p.Path)
				if _, err := os.Stat(projectPath); err == nil {
					p.FolderExists = true
				} else {
					p.FolderExists = false
				}
				ws.Projects = append(ws.Projects, p)
			}
		}
	}

	_, workspacesLocation := getWorkspacesConfigFileLocation(root)
	if workspacesLocation != nil {
		if workspacesLocation.HasDuplicate {
			legacyPath := filepath.Join(root, WorkspacesFileName)
			slog.Warn("Duplicate workspaces file found - using .gws/workspaces.gws, please remove the legacy file",
				"legacy", legacyPath,
				"used", workspacesLocation.Path)
		}

		childRefs, err := parseWorkspacesFile(root)
		if err != nil {
			slog.Warn("Failed to read workspaces", "path", root, "err", err)
		} else {
			for _, childRepository := range childRefs {
				if childRepository.Path == "." { // skip current workspace already added
					continue
				}
				nextRootPath := filepath.Join(root, childRepository.Path)
				if _, err := os.Stat(nextRootPath); err == nil {
					childRepository.FolderExists = true

					if l.recursive {
						resolved, err := l.loadRecursive(nextRootPath, depth+1)
						if err != nil {
							childRepository.Error = err
						} else if resolved != nil {
							//childRepository.Root = resolved.Root
							//childRepository.Projects = resolved.Projects
							//childRepository.Children = resolved.Children
							ws.Children = append(ws.Children, resolved)
						}
					}
				} else {
					ws.Children = append(ws.Children, childRepository)
				}

			}
		}
	}

	slog.Debug(" > gws2 > Loaded workspace", "path", root, "projects", len(ws.Projects), "children", len(ws.Children))
	return ws, nil
}

// FindRoot searches for the root of a workspace starting from the current
// working directory. It traverses parent directories until it finds a
// projects or workspaces configuration file.
//
// # Errors
//
// Returns an error if no workspace root could be located in the current
// or any parent directory.
//
// Example:
//
//	ws, err := FindRoot()
//	if err != nil {
//	    log.Fatal(err)
//	}
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
		hasProjects := hasProjectsFile(dir) || hasProjectsFileInConfigDir(dir)
		hasWorkspaces := hasWorkspacesFile(dir) || hasWorkspacesFileInConfigDir(dir)

		if hasProjects || hasWorkspaces {
			cachedRootDir = dir
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no workspace found (no %s or %s/%s file found in current or parent directories)",
				ProjectsFileName, ConfigDirName, "projects.gws")
		}
		dir = parent
	}
}
