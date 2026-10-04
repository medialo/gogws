package config

import (
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/medialo/gogws/internal/gws2"
	"golang.org/x/term"
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

	// todo is -d use to find root or if -d is present, is considered as root without check
	root, err := gws2.FindRoot()
	if err != nil {
		return err
	}
	if workingDir != "" {
		root = workingDir
	}

	runContext = &RunContext{
		WorkspaceRoot:    root,
		IsInteractive:    term.IsTerminal(int(os.Stdout.Fd())),
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

func GetRunContext() *RunContext {
	runContextMu.RLock()
	defer runContextMu.RUnlock()
	return runContext
}
