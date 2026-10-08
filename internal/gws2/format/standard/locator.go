package standard

import (
	"log/slog"
	"os"
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

func locate(root string) fileLocation {
	var found *fileLocation
	var ignored []string
	for _, dir := range []string{filepath.Join(root, gws2.ConfigDirName), root} {
		for _, enc := range encodings {
			path := filepath.Join(dir, FileBaseName+"."+enc.extension())
			if !fileExists(path) {
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
		return fileLocation{Path: filepath.Join(root, gws2.ConfigDirName, FileBaseName+"."+encodings[0].extension()), Encoding: encodings[0]}
	}
	found.Ignored = ignored
	return *found
}

func (l fileLocation) warnIgnored() {
	if len(l.Ignored) > 0 {
		slog.Warn("Several "+FileBaseName+" files found, please keep only one", "used", l.Path, "ignored", l.Ignored)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
