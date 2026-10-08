package standard

import (
	"log/slog"
	"path/filepath"

	"github.com/medialo/gogws/internal/gws2"
)

const FileBaseName = "workspaces"

type fileLocation struct {
	Path     string
	Exists   bool
	Encoding encoding
	Ignored  []string
}

func locate(files gws2.ConfigFiles) fileLocation {
	var found *fileLocation
	var ignored []string
	for _, inConfigDir := range []bool{true, false} {
		for _, enc := range encodings {
			path, ok := files.Lookup(inConfigDir, FileBaseName+"."+enc.extension())
			if !ok {
				continue
			}
			if found == nil {
				found = &fileLocation{Path: path, Exists: true, Encoding: enc}
				continue
			}
			ignored = append(ignored, path)
		}
	}
	if found == nil {
		return defaultLocation(files.Dir)
	}
	found.Ignored = ignored
	return *found
}

func defaultLocation(dir string) fileLocation {
	return fileLocation{Path: filepath.Join(dir, gws2.ConfigDirName, FileBaseName+"."+encodings[0].extension()), Encoding: encodings[0]}
}

func (l fileLocation) warnIgnored() {
	if len(l.Ignored) > 0 {
		slog.Warn("Several "+FileBaseName+" files found, please keep only one", "used", l.Path, "ignored", l.Ignored)
	}
}
