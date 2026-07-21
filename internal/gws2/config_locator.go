package gws2

import "path/filepath"

type FolderModer int

const (
	ProjectsMode FolderModer = iota
	WorkspacesMode
)

type FileLocation struct {
	Path         string
	IsConfigDir  bool
	HasDuplicate bool
}

func getConfigFileLocation(root string, mode FolderModer) (bool, *FileLocation) {
	switch mode {
	case ProjectsMode:
		return getProjectsConfigFileLocation(root)
	case WorkspacesMode:
		return getWorkspacesConfigFileLocation(root)
	default:
		return false, nil
	}
}

// getProjectsConfigFileLocation checks for the presence of the projects configuration
// file in both the config directory and the legacy location.
// It returns a FileLocation struct indicating where the file is located and whether there are duplicates.
func getProjectsConfigFileLocation(root string) (bool, *FileLocation) {
	// Check for projects file in config directory first, then legacy location
	// <root>/.gws/projects.gws
	configDirPath := filepath.Join(root, ConfigDirName, ProjectsFileName)
	// <root>/projects.gws
	legacyPath := filepath.Join(root, ProjectsFileName)

	hasConfigDir := hasFile(filepath.Join(root, ConfigDirName), ProjectsFileName)
	hasLegacy := hasFile(root, ProjectsFileName)

	if hasConfigDir {
		return false, &FileLocation{
			Path:         configDirPath,
			IsConfigDir:  true,
			HasDuplicate: hasLegacy,
		}
	}
	if hasLegacy {
		return false, &FileLocation{
			Path:         legacyPath,
			IsConfigDir:  false,
			HasDuplicate: false,
		}
	}
	return true, &FileLocation{
		Path:         configDirPath,
		IsConfigDir:  true,
		HasDuplicate: false,
	}
}

func getWorkspacesConfigFileLocation(root string) (bool, *FileLocation) {
	// Check for workspaces file in config directory first, then legacy location
	// <root>/.gws/workspaces.gws
	configDirPath := filepath.Join(root, ConfigDirName, WorkspacesFileName)
	// <root>/workspaces.gws
	legacyPath := filepath.Join(root, WorkspacesFileName)

	hasConfigDir := hasFile(filepath.Join(root, ConfigDirName), WorkspacesFileName)
	hasLegacy := hasFile(root, WorkspacesFileName)

	if hasConfigDir {
		return false, &FileLocation{
			Path:         configDirPath,
			IsConfigDir:  true,
			HasDuplicate: hasLegacy,
		}
	}
	if hasLegacy {
		return false, &FileLocation{
			Path:         legacyPath,
			IsConfigDir:  false,
			HasDuplicate: false,
		}
	}
	return true, &FileLocation{
		Path:         configDirPath,
		IsConfigDir:  true,
		HasDuplicate: false,
	}
}
