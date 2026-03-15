package initcmd

import (
	"fmt"
	"gogws/internal/config"
	"gogws/internal/git"
	"gogws/internal/gitignore"
	"gogws/internal/gws"
	"gogws/internal/ui/cli"
	"log/slog"
	"os"

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
		fileLocation, err := gws.DeleteWorkspacesFile(cfg.WorkspaceRoot)

		if fileLocation != "" {
			fmt.Println(renderer.RenderWarning("Removing workspaces configuration file..."))
			if err != nil {
				return fmt.Errorf("failed to remove existing %s: %w", fileLocation, err)
			}
			fmt.Println(renderer.RenderSuccess(fmt.Sprintf("Workspaces configuration file removed")))
		} else {
			fmt.Println(renderer.RenderError(fmt.Sprintf("%s already exists. Use --reset to reinitialize", gws.WorkspacesFileName)))
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

	for _, ws := range selectedWorkspaces {
		err = gws.AddWorkspace(workspaceRoot, &gws.Workspace{
			Path: ws.Path,
			Root: ws.Path,
			Name: ws.Name,
			Remote: gws.Remote{
				Name: ws.RemoteName,
				URL:  ws.RemoteURL,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to add workspace %s: %w", ws.Path, err)
		}

		//var line string
		//if ws.RemoteURL != "" {
		//	line = fmt.Sprintf("%s | %s %s\n", ws.Path, ws.RemoteURL, ws.RemoteName)
		//} else {
		//	line = fmt.Sprintf("# %s (no remote configured)\n", ws.Path)
		//}
		//if _, err := file.WriteString(line); err != nil {
		//	return fmt.Errorf("failed to write to %s: %w", workspacesFile, err)
		//}
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
