package add

import (
	"fmt"
	"path/filepath"

	"github.com/medialo/gogws/internal/gws2"

	"charm.land/huh/v2"
)

// promptRepoDetails fills in gitURL and folderName from args if both were
// given on the command line, otherwise prompts interactively for whichever
// is missing.
func promptRepoDetails(args []string) (gitURL, folderName string, err error) {
	if len(args) > 0 {
		gitURL = args[0]
	}
	if len(args) > 1 {
		folderName = args[1]
	}

	if gitURL == "" {
		if err := huh.NewInput().Title("Git url of the repository to add").Value(&gitURL).Run(); err != nil {
			return "", "", err
		}
	}
	if gitURL == "" {
		return "", "", fmt.Errorf("a git url is required")
	}

	if folderName == "" {
		if err := huh.NewInput().Title("Folder name").Value(&folderName).Run(); err != nil {
			return "", "", err
		}
	}
	if folderName == "" {
		return "", "", fmt.Errorf("a folder name is required")
	}

	return gitURL, folderName, nil
}

// checkNotAlreadyKnown returns an error if relativePath is already a known
// project or workspace under ws's root.
func checkNotAlreadyKnown(ws *gws2.Workspace, relativePath string) error {
	absPath := filepath.Join(ws.AbsolutePath, relativePath)
	if existing, ok := ws.Index().Get(absPath); ok {
		return fmt.Errorf("%q is already registered as a %s (%s)", existing.GetName(), existing.GetType(), existing.GetPath())
	}
	return nil
}
