package gws2

import (
	"fmt"
	"os"
)

func DeleteProjectsFile(workspaceRoot string) (string, error) {
	_, location := getProjectsConfigFileLocation(workspaceRoot)
	if _, err := os.Stat(location.Path); os.IsNotExist(err) {
		return "", nil
	}
	if err := os.Remove(location.Path); err != nil {
		return location.Path, fmt.Errorf("failed to delete projects file: %w", err)
	}
	return location.Path, nil
}

func DeleteWorkspacesFile(workspaceRoot string) (string, error) {
	_, location := getWorkspacesConfigFileLocation(workspaceRoot)
	if _, err := os.Stat(location.Path); os.IsNotExist(err) {
		return "", nil
	}
	if err := os.Remove(location.Path); err != nil {
		return location.Path, fmt.Errorf("failed to delete workspaces file: %w", err)
	}
	return location.Path, nil
}
