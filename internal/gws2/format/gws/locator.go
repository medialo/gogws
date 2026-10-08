package gws

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/medialo/gogws/internal/gws2"
)

type fileLocation struct {
	Path    string
	Exists  bool
	Ignored string
}

func locateProjectsFile(root string) fileLocation {
	return locate(root, ProjectsFileNameInDir, ProjectsFileName)
}

func locateWorkspacesFile(root string) fileLocation {
	return locate(root, WorkspacesFileNameInDir, WorkspacesFileName)
}

func locate(root, inDirName, legacyName string) fileLocation {
	configDirPath := filepath.Join(root, gws2.ConfigDirName, inDirName)
	legacyPath := filepath.Join(root, legacyName)

	hasConfigDir := fileExists(configDirPath)
	hasLegacy := fileExists(legacyPath)

	if hasConfigDir {
		location := fileLocation{Path: configDirPath, Exists: true}
		if hasLegacy {
			location.Ignored = legacyPath
		}
		return location
	}
	if hasLegacy {
		return fileLocation{Path: legacyPath, Exists: true}
	}
	return fileLocation{Path: configDirPath, Exists: false}
}

func (l fileLocation) warnIgnored() {
	if l.Ignored != "" {
		slog.Warn("Duplicate file found, please remove the legacy file", "legacy", l.Ignored, "used", l.Path)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
