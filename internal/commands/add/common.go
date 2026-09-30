package add

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/ui/prompt"

	"charm.land/huh/v2"
)

func promptRepoDetails(ws *gws2.Workspace, args []string) (gitURL, folderName string, err error) {
	if len(args) > 0 {
		gitURL = args[0]
	}
	if len(args) > 1 {
		folderName = args[1]
	}

	if gitURL == "" {
		if err := prompt.RunField(huh.NewInput().Title("Git url of the repository to add").Value(&gitURL)); err != nil {
			return "", "", err
		}
	}
	if gitURL == "" {
		return "", "", fmt.Errorf("a git url is required")
	}

	if folderName != "" {
		if err := checkNotAlreadyKnown(ws, folderName); err != nil {
			return "", "", err
		}
		return gitURL, folderName, nil
	}

	suggestion := suggestFolderName(gitURL)
	if suggestion != "" && checkNotAlreadyKnown(ws, suggestion) != nil {
		suggestion = ""
	}

	field := huh.NewInput().
		Title("Folder name").
		Value(&folderName).
		Validate(func(s string) error {
			name := s
			if name == "" {
				name = suggestion
			}
			if name == "" {
				return fmt.Errorf("a folder name is required")
			}
			return checkNotAlreadyKnown(ws, name)
		})
	if suggestion != "" {
		field = field.Placeholder(suggestion)
	}

	if err := prompt.RunField(field); err != nil {
		return "", "", err
	}
	if folderName == "" {
		folderName = suggestion
	}
	if folderName == "" {
		return "", "", fmt.Errorf("a folder name is required")
	}

	return gitURL, folderName, nil
}

func suggestFolderName(gitURL string) string {
	name := strings.TrimRight(gitURL, "/")
	name = strings.TrimSuffix(name, ".git")
	if idx := strings.LastIndex(name, "/"); idx != -1 {
		return name[idx+1:]
	}
	if idx := strings.LastIndex(name, ":"); idx != -1 {
		return name[idx+1:]
	}
	return name
}

func checkNotAlreadyKnown(ws *gws2.Workspace, relativePath string) error {
	absPath := filepath.Join(ws.AbsolutePath, relativePath)
	if existing, ok := ws.Index().Get(absPath); ok {
		kind := "entry"
		switch existing.(type) {
		case *gws2.Project:
			kind = "project"
		case *gws2.Workspace:
			kind = "workspace"
		}
		return fmt.Errorf("%q is already registered as a %s (%s)", existing.GetName(), kind, existing.GetPath())
	}
	return nil
}
