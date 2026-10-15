package add

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/gws2"
)

func entryPath(cfg *config.RunContext, name string) (string, error) {
	if !cfg.Global || gws2.IsAnchoredPath(name) {
		return name, nil
	}
	abs, err := filepath.Abs(name)
	if err != nil {
		return "", fmt.Errorf("failed to resolve %q: %w", name, err)
	}
	return abbreviateHome(abs), nil
}

func discoveryRoot(cfg *config.RunContext) (string, error) {
	if !cfg.Global {
		return cfg.WorkspaceRoot, nil
	}
	return os.UserHomeDir()
}

func abbreviateHome(abs string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return abs
	}
	rel, err := filepath.Rel(home, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return abs
	}
	if rel == "." {
		return "~"
	}
	return "~/" + filepath.ToSlash(rel)
}
