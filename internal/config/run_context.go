package config

import (
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/medialo/gogws/internal/gws2/loader"
	"github.com/medialo/gogws/internal/interactive"
)

type RunContext struct {
	WorkspaceRoot    string
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

func Initialize(prefs *Preferences, onlyChanges bool, workingDir string) error {
	slog.Debug("Initializing run context...")

	runContextMu.Lock()
	defer runContextMu.Unlock()

	if runContext != nil {
		slog.Debug("Run context already initialized")
		return nil
	}

	root, err := resolveWorkspaceRoot(workingDir)
	if err != nil {
		return err
	}

	runContext = &RunContext{
		WorkspaceRoot:    root,
		IsInteractive:    interactive.Enabled(),
		OnlyChanges:      onlyChanges,
		Parallel:         prefs.Parallel.Value,
		Format:           prefs.Format.Value,
		Theme:            prefs.Theme.Value,
		StopOnError:      prefs.StopOnError.Value,
		ProviderCacheTTL: prefs.ProviderCacheTTL.Value,
	}

	slog.Debug("Run context initialized", "runContext", runContext)

	return nil
}

func resolveWorkspaceRoot(workingDir string) (string, error) {
	if workingDir != "" {
		return filepath.Abs(workingDir)
	}
	return loader.FindRoot()
}

func GetRunContext() *RunContext {
	runContextMu.RLock()
	defer runContextMu.RUnlock()
	return runContext
}
