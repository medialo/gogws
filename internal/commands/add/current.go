package add

import (
	"fmt"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/ui/cli"
	"github.com/medialo/gogws/internal/ui/prompt"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

var currentAsWorkspace bool
var currentAsProject bool

func newAddCurrentCommand(getConfig func() *config.RunContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "current",
		Aliases: []string{".", "self"},
		Short:   "Add the current repository to its workspace",
		Long: `Register the git repository of the current directory as a project (default) or a workspace of the enclosing workspace. 
/!\ You must be in the same level as your project's .git folder`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddCurrent(getConfig)
		},
	}

	cmd.Flags().BoolVarP(&currentAsWorkspace, "workspace", "w", false, "Add the repository as a workspace directly (no ask box)")
	cmd.Flags().BoolVarP(&currentAsProject, "project", "p", false, "Add the repository as a project directly (no ask box)")
	return cmd
}

func runAddCurrent(getConfig func() *config.RunContext) error {
	if currentAsProject && currentAsWorkspace {
		return fmt.Errorf("--workspace and --project can not be set at the same time")
	}

	cfg := getConfig()
	if cfg == nil {
		return fmt.Errorf("no workspace found (no .projects.gws file)")
	}

	ws, err := gws2.NewFromPath(cfg.WorkspaceRoot).RunDoctor(false).Recursive(false).Load()
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	repo, err := git.DiscoverRepository(cfg.WorkspaceRoot, cwd)
	if err != nil {
		return err
	}
	if repo == nil {
		return fmt.Errorf("current directory is not a git repository inside workspace %s", cfg.WorkspaceRoot)
	}
	if len(repo.Remotes) == 0 {
		return fmt.Errorf("repository %q has no remote", repo.Path)
	}

	if err := checkNotAlreadyKnown(ws, repo.Path); err != nil {
		return err
	}

	asProject := true
	if cfg.IsInteractive && !currentAsWorkspace && !currentAsProject {
		field := huh.NewConfirm().
			Title(fmt.Sprintf("Add %q as", repo.Path)).
			Affirmative("Project").
			Negative("Workspace").
			Value(&asProject)
		if err := prompt.RunField(field); err != nil {
			return err
		}
	}

	if currentAsWorkspace {
		asProject = false
	}

	renderer := cli.NewRenderer()

	if asProject {
		ws.AddProject(gws2.NewProject(cfg.WorkspaceRoot, repo.Path, repo.Remotes))
		if err := ws.SaveProjects(); err != nil {
			return fmt.Errorf("failed to add project: %w", err)
		}
		lipgloss.Println(renderer.RenderSuccess(fmt.Sprintf("Added project %q", repo.Path)))
		return nil
	}

	ws.AddWorkspace(gws2.NewChildWorkspace(cfg.WorkspaceRoot, repo.Path, repo.Remotes))
	if err := ws.SaveWorkspace(); err != nil {
		return fmt.Errorf("failed to add workspace: %w", err)
	}
	lipgloss.Println(renderer.RenderSuccess(fmt.Sprintf("Added workspace %q", repo.Path)))
	return nil
}
