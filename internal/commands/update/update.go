package update

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"gogws/internal/config"
	"gogws/internal/engine"
	"gogws/internal/git"
	"gogws/internal/gws"
	"gogws/internal/hooks"
	"gogws/internal/ui/cli"

	"github.com/spf13/cobra"
)

var (
	skipProjects   bool
	skipWorkspaces bool
)

func NewCommand(getConfig func() *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Clone all missing repositories and workspaces",
		Long: `Clone all repositories defined in .projects.gws and workspaces defined 
in .workspaces.gws that are not yet present in the workspace.

Use --skip-projects to only clone workspaces (recursive).
Use --skip-workspaces to only clone projects.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(getConfig)
		},
	}

	cmd.Flags().BoolVar(&skipProjects, "skip-projects", false, "skip cloning projects, only clone workspaces")
	cmd.Flags().BoolVar(&skipWorkspaces, "skip-workspaces", false, "skip cloning workspaces, only clone projects")

	return cmd
}

func runUpdate(getConfig func() *config.Config) error {
	cfg := getConfig()
	if cfg == nil {
		return fmt.Errorf("no workspace found (no .projects.gws file)")
	}

	if err := hooks.PreUpdate(cfg.WorkspaceRoot); err != nil {
		return fmt.Errorf("pre-update hook failed: %w", err)
	}

	slog.Debug(fmt.Sprintf("Running update command in workspace: %s", cfg.WorkspaceRoot))
	ws, err := gws.New(cfg.WorkspaceRoot).Load()
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	renderer := cli.NewRenderer()
	output := engine.NewOutputHandler(renderer, false)
	var clonedProjects []string

	if !skipWorkspaces && len(ws.Children) > 0 {
		result := cloneWorkspaces(cfg.WorkspaceRoot, ws, cfg.Parallel, cfg.StopOnError)
		output.RenderSummary(result, "Cloned workspaces")
	}

	if !skipProjects {
		missingProjects := ws.MissingProjects()
		if len(missingProjects) == 0 {
			fmt.Println(renderer.RenderSuccess("All projects are already cloned"))
		} else {
			fmt.Println(renderer.RenderInfo(fmt.Sprintf("Cloning %d missing projects...", len(missingProjects))))

			result := cloneProjects(cfg.WorkspaceRoot, missingProjects, cfg.Parallel, cfg.StopOnError)
			output.RenderSummary(result, "Cloned projects")

			for _, r := range result.Succeeded() {
				clonedProjects = append(clonedProjects, r.Command.RepoName)
			}
		}
	}

	if err := hooks.PostUpdate(cfg.WorkspaceRoot, clonedProjects); err != nil {
		return fmt.Errorf("post-update hook failed: %w", err)
	}

	return nil
}

func cloneWorkspaces(workspaceRoot string, ws *gws.Workspace, parallel int, stopOnError bool) *engine.ExecuteResult {
	toClone := ws.MissingWorkspaces()
	if len(toClone) == 0 {
		return engine.NewExecuteResult()
	}

	commands := make([]engine.RepoCommand, 0, len(toClone))

	for _, child := range toClone {
		remotes := []git.Remote{{Name: child.Remote.Name, URL: child.Remote.URL}}
		wsRoot := workspaceRoot
		childPath := child.Path

		cmd := engine.NewCustomCommand(
			filepath.Join(workspaceRoot, child.Path),
			child.Path,
			func() (string, error) {
				return "", git.CloneWorkspace(wsRoot, childPath, remotes)
			},
		)
		commands = append(commands, cmd)
	}

	return engine.Execute(commands, engine.ExecuteOptions{
		Parallel:    parallel,
		StopOnError: stopOnError,
	})
}

func cloneProjects(workspaceRoot string, toClone []gws.Project, parallel int, stopOnError bool) *engine.ExecuteResult {
	commands := make([]engine.RepoCommand, 0, len(toClone))

	for _, p := range toClone {
		remotes := git.ToGitRemotes(p.Remotes)
		wsRoot := workspaceRoot
		projectPath := p.Path

		cmd := engine.NewCustomCommand(
			filepath.Join(workspaceRoot, p.Path),
			p.Path,
			func() (string, error) {
				return "", git.CloneWorkspace(wsRoot, projectPath, remotes)
			},
		)
		commands = append(commands, cmd)
	}

	return engine.Execute(commands, engine.ExecuteOptions{
		Parallel:    parallel,
		StopOnError: stopOnError,
	})
}
