package add

import (
	"context"
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/ui/cli"

	"github.com/spf13/cobra"
)

var autoCloneProject bool

func newAddProjectCommand(getConfig func() *config.RunContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project [git url] [foldername]",
		Short: "Add a project to the current workspace",
		Long:  `Add a project to the .projects.gws file, creating it if needed.`,
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddProject(getConfig, args)
		},
	}

	cmd.Flags().BoolVar(&autoCloneProject, "auto-clone", false, "Clone the repository immediately after adding it to the workspace")
	return cmd
}

func runAddProject(getConfig func() *config.RunContext, args []string) error {
	cfg := getConfig()
	if cfg == nil {
		return fmt.Errorf("no workspace found (no .projects.gws file)")
	}

	ws, err := gws2.NewFromPath(cfg.WorkspaceRoot).RunDoctor(false).Recursive(false).Load()
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	gitURL, folderName, err := promptRepoDetails(ws, args)
	if err != nil {
		return err
	}

	project := gws2.NewProject(cfg.WorkspaceRoot, folderName, git.OriginRemotes(gitURL))

	ws.AddProject(project)
	if err := ws.SaveProjects(); err != nil {
		return fmt.Errorf("failed to add project: %w", err)
	}

	renderer := cli.NewRenderer()
	lipgloss.Println(renderer.RenderSuccess(fmt.Sprintf("Added project %q -> %s", folderName, gitURL)))

	if autoCloneProject {
		if err := git.Clone(context.Background(), project.GetOriginRemote(), project.GetPath(), nil); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
		lipgloss.Println(renderer.RenderSuccess(fmt.Sprintf("Cloned %s", folderName)))
	}

	return nil
}
