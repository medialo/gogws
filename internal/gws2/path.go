package gws2

import (
	"os"
	"path/filepath"
	"strings"
)

func ResolvePath(rootPath, configPath string) string {
	if expanded, ok := expandHome(configPath); ok {
		return expanded
	}
	if filepath.IsAbs(configPath) {
		return filepath.Clean(configPath)
	}
	return filepath.Join(rootPath, configPath)
}

func IsAnchoredPath(path string) bool {
	if _, ok := expandHome(path); ok {
		return true
	}
	return filepath.IsAbs(path)
}

func expandHome(path string) (string, bool) {
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, `~\`) {
		return "", false
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(home, path[1:]), true
}
