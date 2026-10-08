package gws

import (
	"log/slog"
	"path/filepath"

	"github.com/medialo/gogws/internal/gws2"
)

type fileLocation struct {
	Path    string
	Exists  bool
	Ignored string
}

func locateProjectsFile(files gws2.ConfigFiles) fileLocation {
	return locate(files, ProjectsFileNameInDir, ProjectsFileName)
}

func locateWorkspacesFile(files gws2.ConfigFiles) fileLocation {
	return locate(files, WorkspacesFileNameInDir, WorkspacesFileName)
}

func locate(files gws2.ConfigFiles, inDirName, legacyName string) fileLocation {
	configDirPath, hasConfigDir := files.Lookup(true, inDirName)
	legacyPath, hasLegacy := files.Lookup(false, legacyName)

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
	return defaultLocation(files.Dir, inDirName)
}

func defaultLocation(dir, inDirName string) fileLocation {
	return fileLocation{Path: filepath.Join(dir, gws2.ConfigDirName, inDirName)}
}

func (l fileLocation) warnIgnored() {
	if l.Ignored != "" {
		slog.Warn("Duplicate file found, please remove the legacy file", "legacy", l.Ignored, "used", l.Path)
	}
}
