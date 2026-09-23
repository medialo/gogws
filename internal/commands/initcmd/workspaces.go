package initcmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gitignore"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/ui/cli"

	"charm.land/huh/v2"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var (
	workspacesGitignore    bool
	resetWorkspacesGwsFile bool
)

func newWorkspacesCommand(getConfig func() *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspaces",
		Short: "Interactively configure sub-workspaces",
		Long: `Scan subdirectories and interactively select which ones should be
configured as sub-workspaces. For each selected directory, you can provide
a git remote URL.

Creates a .gws/workspaces.gws file with the configured workspaces.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return preRunInitWorkspaces(getConfig)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInitWorkspaces()
		},
	}

	cmd.Flags().BoolVar(&workspacesGitignore, "reset", false, "reset existing workspaces.gws file if it exists")

	return cmd
}

func preRunInitWorkspaces(getConfig func() *config.Config) error {
	if resetWorkspacesGwsFile {
		cfg := getConfig()
		slog.Debug("Resetting workspaces.gws file", "resetWorkspacesGwsFile", resetWorkspacesGwsFile)
		fileLocation, err := gws2.DeleteWorkspacesFile(cfg.WorkspaceRoot)

		if fileLocation != "" {
			fmt.Println(renderer.RenderWarning("Removing workspaces configuration file..."))
			if err != nil {
				return fmt.Errorf("failed to remove existing %s: %w", fileLocation, err)
			}
			fmt.Println(renderer.RenderSuccess(fmt.Sprintf("Workspaces configuration file removed")))
		} else {
			fmt.Println(renderer.RenderError(fmt.Sprintf("%s already exists. Use --reset to reinitialize", gws2.WorkspacesFileName)))
		}
	}
	return nil
}

func runInitWorkspaces() error {
	workspaceRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	renderer := cli.NewRenderer()
	fmt.Println(renderer.RenderInfo("Scanning current folder for git repositories..."))

	discoveredGitRepo, err := git.DiscoverRepositories(workspaceRoot, 1)
	if err != nil {
		return fmt.Errorf("failed to discover git repositories: %w", err)
	}

	if len(discoveredGitRepo) == 0 {
		fmt.Println(renderer.RenderWarning("No git repositories found in subdirectories"))
		return nil
	}

	var selectedWorkspaces []workspaceEntry
	opts := lo.Map(discoveredGitRepo, func(item git.DiscoveredRepo, index int) huh.Option[workspaceEntry] {
		var remoteURL, remoteName string
		if len(item.Remotes) > 0 {
			slog.Debug("Repository has no remotes", "path", item.Path)
			remoteURL = item.Remotes[0].URL
			remoteName = item.Remotes[0].Name
		} else {
			slog.Debug("Repository has remotes", "path", item.Path)
			remoteURL = ""
			remoteName = ""
		}
		return huh.Option[workspaceEntry]{
			Value: workspaceEntry{
				Path:       item.Path,
				Name:       item.Path,
				RemoteURL:  remoteURL,
				RemoteName: remoteName,
			},
			Key: item.Path,
		}
	})

	huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[workspaceEntry]().
				Title("Found subdirectories:").
				Options(opts...).
				Value(&selectedWorkspaces))).Run()

	if len(selectedWorkspaces) == 0 {
		fmt.Println(renderer.RenderWarning("No workspaces configured"))
		return nil
	}

	//gwsDir := filepath.Join(workspaceRoot, gws.ConfigDirName)
	//if err := os.MkdirAll(gwsDir, 0755); err != nil {
	//	return fmt.Errorf("failed to create %s directory: %w", gws.ConfigDirName, err)
	//}
	//
	//workspacesFile := filepath.Join(gwsDir, gws.WorkspacesFileName)
	//
	//file, err := os.Create(workspacesFile)
	//if err != nil {
	//	return fmt.Errorf("failed to create %s: %w", workspacesFile, err)
	//}
	//defer file.Close()

	root, err := gws2.NewFromPath(workspaceRoot).RunDoctor(false).Recursive(false).Load()
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	for _, ws := range selectedWorkspaces {
		var child *gws2.Workspace
		if ws.RemoteURL != "" {
			remoteName := ws.RemoteName
			if remoteName == "" {
				remoteName = "origin"
			}
			child = gws2.NewChildWorkspace(workspaceRoot, ws.Path, &git.Remote{Name: remoteName, URL: ws.RemoteURL})
		} else {
			child = gws2.NewFolderWorkspace(workspaceRoot, ws.Path)
		}
		root.AddWorkspace(child)
	}

	if err := root.SaveWorkspace(); err != nil {
		return fmt.Errorf("failed to save workspaces: %w", err)
	}

	fmt.Println()
	fmt.Println(renderer.RenderSuccess(fmt.Sprintf("Created %d workspaces in workpaces configuration file", len(selectedWorkspaces))))

	if workspacesGitignore {
		if err := gitignore.EnsureGWSSection(workspaceRoot); err != nil {
			fmt.Println(renderer.RenderWarning(fmt.Sprintf("Failed to generate .gitignore: %v", err)))
		} else {
			fmt.Println(renderer.RenderSuccess("Generated .gitignore"))
		}
	}

	return nil
}

type workspaceEntry struct {
	Path       string
	Name       string
	RemoteURL  string
	RemoteName string
}
