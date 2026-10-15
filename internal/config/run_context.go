package config

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/medialo/gogws/internal/gws2/loader"
	"github.com/medialo/gogws/internal/interactive"
)

type RunContext struct {
	WorkspaceRoot    string
	Global           bool
	IsInteractive    bool
	OnlyChanges      bool
	Parallel         int
	Format           string
	Theme            string
	StopOnError      bool
	ProviderCacheTTL time.Duration
}

var (
	runContext   *RunContext
	runContextMu sync.RWMutex
)

func Initialize(prefs *Preferences, onlyChanges bool, workingDir string, mode loader.WorkspaceMode, allowGlobal bool) error {
	slog.Debug("Initializing run context...")

	runContextMu.Lock()
	defer runContextMu.Unlock()

	if runContext != nil {
		slog.Debug("Run context already initialized")
		return nil
	}

	root, global, err := resolveWorkspaceRoot(workingDir, mode, allowGlobal && prefs.GlobalFallback.value)
	if err != nil {
		return err
	}

	runContext = &RunContext{
		WorkspaceRoot:    root,
		Global:           global,
		IsInteractive:    interactive.Enabled(),
		OnlyChanges:      onlyChanges,
		Parallel:         prefs.Parallel.value,
		Format:           prefs.Format.value,
		Theme:            prefs.Theme.value,
		StopOnError:      prefs.StopOnError.value,
		ProviderCacheTTL: prefs.ProviderCacheTTL.value,
	}

	slog.Debug("Run context initialized", "runContext", runContext)

	return nil
}

func resolveWorkspaceRoot(workingDir string, mode loader.WorkspaceMode, globalFallback bool) (string, bool, error) {
	if mode == loader.ModeGlobal {
		root, err := globalWorkspaceRoot()
		return root, true, err
	}
	if mode == loader.ModeNearest && workingDir != "" {
		root, err := filepath.Abs(workingDir)
		return root, false, err
	}

	start := workingDir
	if start == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", false, err
		}
		start = cwd
	}

	root, err := loader.FindRoot(start, mode)
	if err == nil || !errors.Is(err, loader.ErrNoWorkspace) || !globalFallback || !globalWorkspaceExists() {
		return root, false, err
	}

	globalRoot, rootErr := globalWorkspaceRoot()
	if rootErr != nil {
		return "", false, err
	}
	slog.Debug("No workspace found, falling back to the global workspace", "start", start)
	return globalRoot, true, nil
}

func globalWorkspaceRoot() (string, error) {
	path, err := GetGlobalWorkspacesPath()
	if err != nil {
		return "", err
	}
	return filepath.Dir(path), nil
}

func globalWorkspaceExists() bool {
	path, err := GetGlobalWorkspacesPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func (c *RunContext) NewLoader() *loader.Loader {
	if c.Global {
		if path, err := GetGlobalWorkspacesPath(); err == nil {
			return loader.NewGlobal(c.WorkspaceRoot, path)
		}
	}
	return loader.NewFromPath(c.WorkspaceRoot)
}

func GetRunContext() *RunContext {
	runContextMu.RLock()
	defer runContextMu.RUnlock()
	return runContext
}
