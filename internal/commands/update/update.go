package update

import (
	"context"
	"fmt"
	"gogws/internal/config"
	"gogws/internal/engine"
	"gogws/internal/git"
	"gogws/internal/gws"
	"gogws/internal/hooks"
	"gogws/internal/ui/cli"
	engineui "gogws/internal/ui/engineui"
	"log/slog"

	"github.com/spf13/cobra"
)

var (
	skipProjects   bool
	skipWorkspaces bool
	recursive      bool
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
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Set update to recursive mode (clone all workspaces and sub-projects)")

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
	var clonedProjects []string

	if !skipWorkspaces && len(ws.Children) > 0 {
		result := cloneWorkspaces(cfg.WorkspaceRoot, ws, cfg.Parallel, cfg.StopOnError, cfg.IsInteractive)
		if !cfg.IsInteractive {
			renderSummary(renderer, result, "Cloned workspaces")
		}
	}

	if !skipProjects {
		missingProjects := ws.MissingProjects()
		if len(missingProjects) == 0 {
			fmt.Println(renderer.RenderSuccess("All projects are already cloned"))
		} else {
			fmt.Println(renderer.RenderInfo(fmt.Sprintf("Cloning %d missing projects...", len(missingProjects))))

			result := cloneProjects(cfg.WorkspaceRoot, missingProjects, cfg.Parallel, cfg.StopOnError, cfg.IsInteractive)
			if !cfg.IsInteractive {
				renderSummary(renderer, result, "Cloned projects")
			}

			for _, label := range result.SuccessLabels() {
				clonedProjects = append(clonedProjects, label)
			}
		}
	}

	if err := hooks.PostUpdate(cfg.WorkspaceRoot, clonedProjects); err != nil {
		return fmt.Errorf("post-update hook failed: %w", err)
	}

	return nil
}

// todo check if engine bien placé
func cloneWorkspaces(workspaceRoot string, ws *gws.Workspace, parallel int, stopOnError bool, isInteractive bool) *engine.ExecuteResult {
	toClone := ws.MissingWorkspaces()
	if len(toClone) == 0 {
		return engine.NewNoExecutionResult()
	}

	jobs := make([]engine.Job, 0, len(toClone))

	for _, child := range toClone {
		remotes := []git.Remote{{Name: child.Remote.Name, URL: child.Remote.URL}}
		wsRoot := workspaceRoot
		childPath := child.Path

		jobs = append(jobs, engine.Job{
			JobNameId: child.Path,
			Fn: func(ctx context.Context, notify engine.Notify) error {
				return git.CloneWorkspace(ctx, wsRoot, childPath, remotes, engine.WrapRunner(notify))
			},
		})
	}

	return runJobs(jobs, parallel, stopOnError, isInteractive)
}

func cloneProjects(workspaceRoot string, toClone []gws.Project, maxParallel int, stopOnError bool, isInteractive bool) *engine.ExecuteResult {
	jobs := make([]engine.Job, 0, len(toClone))

	for _, p := range toClone {
		remotes := git.ToGitRemotes(p.Remotes)
		projectPath := p.Path

		jobs = append(jobs, engine.Job{
			JobNameId: p.Path,
			Fn: func(ctx context.Context, notify engine.Notify) error {
				return git.Clone(ctx, projectPath, remotes, engine.WrapRunner(notify))
			},
		})
	}

	return runJobs(jobs, maxParallel, stopOnError, isInteractive)
}

func runJobs(jobs []engine.Job, maxParallel int, stopOnError bool, isInteractive bool) *engine.ExecuteResult {
	opts := engine.DefaultOptions().
		WithParallel(maxParallel).
		WithStopOnError(stopOnError)

	eng := engine.NewEngine(opts)
	events, resultCh := eng.RunJobs(context.Background(), jobs)

	if isInteractive {
		if err := engineui.Run(events, opts.Parallel, len(jobs)); err != nil {
			slog.Error("UI error", "error", err)
		}
	} else {
		engine.ConsumeVerbose(events)
	}

	return <-resultCh
}

func renderSummary(renderer *cli.Renderer, result *engine.ExecuteResult, action string) {
	if result.HasErrors() {
		for _, r := range result.Failed() {
			renderer.RenderError(fmt.Sprintf("%s: %v", r.JobId, r.Error))
		}
	}
	renderer.RenderSuccess(fmt.Sprintf("%s %d repositories", action, result.SuccessCount()))
	if result.SkippedCount() > 0 {
		renderer.RenderWarning(fmt.Sprintf("Skipped %d repositories", result.SkippedCount()))
	}
}
