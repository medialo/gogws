package gws2

import (
	"os"
	"path/filepath"
)

// hasFile checks if a file with the given name exists in the specified root directory.
// It returns true if the file exists, and false otherwise.
func hasFile(root, fileName string) bool {
	path := filepath.Join(root, fileName)
	_, err := os.Stat(path)
	return err == nil
}

func hasProjectsFile(root string) bool {
	return hasFile(root, ProjectsFileName)
}

func hasProjectsFileInConfigDir(root string) bool {
	return hasFile(filepath.Join(root, ConfigDirName), ProjectsFileName)
}

func hasWorkspacesFile(root string) bool {
	return hasFile(root, WorkspacesFileName)
}

func hasWorkspacesFileInConfigDir(root string) bool {
	return hasFile(filepath.Join(root, ConfigDirName), WorkspacesFileName)
}

func ptr[T any](v T) *T {
	return &v
}
