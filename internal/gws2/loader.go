package gws2

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
		maxDepth:  DefaultMaxDepth,
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

func (l *Loader) Load() (*Workspace, error) {
	slog.Debug("Loading workspace loader...", "path", l.root)
	ws, err := l.loadRecursiveInit(l.root)
	// todo check lock, si invalid la command doctor a besoin d'un contexte valide donc loop inifini coté user
	if ws == nil || (l.runDoctor && !ws.IsValid()) {
		return nil, fmt.Errorf("workspace is in invalid state, please run 'gogws doctor' to show diagnostics")
	}
	slog.Debug("Found projects and workspaces", "projects", len(ws.Projects), "workspaces", len(ws.Children), "rootIsGit", ws.IsGitRepository())
	return ws, err
}

func (l *Loader) loadRecursiveInit(rootPath string) (*Workspace, error) {
	ws := NewRootWorkspace(rootPath)
	ws.setIndex(NewIndex())

	return l.loadRecursive(ws, rootPath, 0)
}

func (l *Loader) loadRecursive(wsRootForCurrRecurCall *Workspace, rootPath string, depth int) (*Workspace, error) {
	idx := wsRootForCurrRecurCall.Index()
	if idx.Has(rootPath) {
		slog.Debug("Skipping already visited workspace", "path", rootPath)
		wsRootForCurrRecurCall.Error = fmt.Errorf("workspace already visited (possible cycle): %s", rootPath)
		return wsRootForCurrRecurCall, nil
	}
	idx.put(wsRootForCurrRecurCall)

	if depth > l.maxDepth {
		slog.Warn("Maximum workspace depth reached", "path", rootPath)
		wsRootForCurrRecurCall.Error = fmt.Errorf("maximum workspace depth (%d) reached at: %s", l.maxDepth, rootPath)
		return wsRootForCurrRecurCall, nil
	}

	slog.Debug("Loading workspace", "depth", depth, "path", rootPath)

	_, projectsLocation := getProjectsConfigFileLocation(rootPath)
	if projectsLocation != nil {
		if projectsLocation.HasDuplicate {
			legacyPath := filepath.Join(rootPath, ProjectsFileName)
			slog.Warn("Duplicate projects file found - using .gws/projects.gws, please remove the legacy file",
				"legacy", legacyPath,
				"used", projectsLocation.Path)
		}

		projectsFromFile, err := parseProjectsFile(rootPath)
		if err != nil {
			slog.Warn("Failed to read projects", "path", rootPath, "err", err)
		} else {
			for _, p := range projectsFromFile {
				if _, err := os.Stat(p.AbsolutePath); err == nil {
					p.FolderExists = true
				} else {
					p.FolderExists = false
				}
				wsRootForCurrRecurCall.AddProject(p)
			}
		}
	}

	_, workspacesLocation := getWorkspacesConfigFileLocation(rootPath)
	if workspacesLocation != nil {
		if workspacesLocation.HasDuplicate {
			legacyPath := filepath.Join(rootPath, WorkspacesFileName)
			slog.Warn("Duplicate workspaces file found - using .gws/workspaces.gws, please remove the legacy file",
				"legacy", legacyPath,
				"used", workspacesLocation.Path)
		}

		workspacesFromFile, err := parseWorkspacesFile(rootPath)
		if err != nil {
			slog.Warn("Failed to read workspaces", "path", rootPath, "err", err)
		} else {
			for _, childRepository := range workspacesFromFile {
				if childRepository.AbsolutePath == "." { // skip current workspace already added
					continue
				}
				nextRootPath := childRepository.AbsolutePath
				if _, err := os.Stat(nextRootPath); err == nil {
					childRepository.FolderExists = true

					if l.recursive {
						// The child must carry the shared index before it
						// recurses, so its own cycle check and registration
						// land in the same index as everyone else's.
						childRepository.setIndex(idx)
						resolved, err := l.loadRecursive(childRepository, nextRootPath, depth+1)
						if err != nil {
							childRepository.Error = err
							wsRootForCurrRecurCall.AddWorkspace(childRepository)
						} else if resolved != nil {
							wsRootForCurrRecurCall.AddWorkspace(resolved)
						}
					} else {
						wsRootForCurrRecurCall.AddWorkspace(childRepository)
					}
				} else {
					wsRootForCurrRecurCall.AddWorkspace(childRepository)
				}

			}
		}
	}

	slog.Debug(" > gws2 > Loaded workspace", "path", rootPath, "projects", len(wsRootForCurrRecurCall.Projects), "children", len(wsRootForCurrRecurCall.Children))
	return wsRootForCurrRecurCall, nil
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
