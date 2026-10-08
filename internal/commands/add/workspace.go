package add

import (
	"context"
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/gws2/loader"
	"github.com/medialo/gogws/internal/ui/cli"
	"github.com/medialo/gogws/internal/ui/prompt"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

var (
	autoCloneWorkspace bool
	folderWorkspace    bool
)

func newAddWorkspaceCommand(getConfig func() *config.RunContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace [git url] [foldername]",
		Short: "Add a workspace to the current workspace",
		Long:  `Add a sub-workspace to the .workspaces.gws file, creating it if needed.`,
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddWorkspace(getConfig, args)
		},
	}

	cmd.Flags().BoolVar(&autoCloneWorkspace, "auto-clone", false, "Clone the repository immediately after adding it to the workspace")
	cmd.Flags().BoolVar(&folderWorkspace, "folder", false, "Add a plain folder workspace instead (no git remote); only the folder name is needed")
	return cmd
}

func runAddWorkspace(getConfig func() *config.RunContext, args []string) error {
	cfg := getConfig()
	if cfg == nil {
		return fmt.Errorf("no workspace found (no .projects.gws file)")
	}

	ws, err := loader.NewFromPath(cfg.WorkspaceRoot).RunDoctor(false).Recursive(false).Load()
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	renderer := cli.NewRenderer()

	if folderWorkspace {
		folderName := ""
		if len(args) > 0 {
			folderName = args[0]
		}
		if folderName == "" {
			if err := prompt.RunField(huh.NewInput().Title("Folder name").Value(&folderName)); err != nil {
				return err
			}
		}
		if folderName == "" {
			return fmt.Errorf("a folder name is required")
		}

		if err := checkNotAlreadyKnown(ws, folderName); err != nil {
			return err
		}

		child := gws2.NewFolderWorkspace(cfg.WorkspaceRoot, folderName)
		ws.AddWorkspace(child)
		if err := ws.SaveWorkspace(); err != nil {
			return fmt.Errorf("failed to add workspace: %w", err)
		}

		lipgloss.Println(renderer.RenderSuccess(fmt.Sprintf("Added folder workspace %q", folderName)))
		return nil
	}

	gitURL, folderName, err := promptRepoDetails(ws, args)
	if err != nil {
		return err
	}

	child := gws2.NewChildWorkspace(cfg.WorkspaceRoot, folderName, git.OriginRemotes(gitURL))

	ws.AddWorkspace(child)
	if err := ws.SaveWorkspace(); err != nil {
		return fmt.Errorf("failed to add workspace: %w", err)
	}

	lipgloss.Println(renderer.RenderSuccess(fmt.Sprintf("Added workspace %q -> %s", folderName, gitURL)))

	if autoCloneWorkspace {
		if err := git.CloneWorkspace(context.Background(), child.GetOriginRemote(), child.GetPath(), nil); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
		lipgloss.Println(renderer.RenderSuccess(fmt.Sprintf("Cloned %s", folderName)))
	}

	return nil
}
