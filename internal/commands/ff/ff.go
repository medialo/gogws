package ff

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/gws2/loader"
	"github.com/medialo/gogws/internal/ui/engineui"
	"github.com/samber/lo"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/engine"
	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/hooks"
	"github.com/medialo/gogws/internal/ui/cli"

	"github.com/spf13/cobra"
)

func NewCommand(getConfig func() *config.RunContext) *cobra.Command {
	return &cobra.Command{
		Use:   "ff",
		Short: "Fast-forward pull all repositories",
		Long:  `Fast-forward pull from origin for all repositories (only if fast-forward is possible).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFF(getConfig)
		},
	}
}

func runFF(getConfig func() *config.RunContext) error {
	cfg := getConfig()
	if cfg == nil {
		return fmt.Errorf("no workspace found")
	}

	if err := hooks.PreFF(cfg.WorkspaceRoot); err != nil {
		return fmt.Errorf("pre-ff hook failed: %w", err)
	}

	slog.Debug("Running ff command", "workspace", cfg.WorkspaceRoot)

	ws, err := loader.NewFromPath(cfg.WorkspaceRoot).Recursive(true).Load()
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	projectList := lo.Filter(ws.FlattenProjects(), func(item *gws2.Project, index int) bool {
		return item.FolderExist() && item.IsGitRepository()
	})

	projectHooks, err := hooks.PrepareProjectHooks(ws, hooks.ProjectHooksOptions{Command: "ff", Pre: hooks.HookPreFF, Post: hooks.HookPostFF})
	if err != nil {
		return fmt.Errorf("failed to prepare project hooks: %w", err)
	}

	jobs := make([]engine.Job, 0, len(projectList))
	var skippedJobs []engine.JobResult

	slog.Debug("ff command: creating jobs...", "workspace", cfg.WorkspaceRoot, "nbProjects", len(jobs))
	start := time.Now()
	for _, p := range projectList {
		jobs = append(jobs, engine.Job{
			JobNameId: p.GetPath(),
			Fn: func(ctx context.Context, notify engine.Notify) error {
				notify.Log("Checking if project is cloned...")

				if !git.IsRepository(p.GetPath()) {
					return fmt.Errorf("%s is not a git repository", p.GetPath())
				}

				return projectHooks.Around(ctx, p, notify, func() error {
					notify.Log("Fast-forwarding...")
					return engine.Wrap(git.Pull(p.GetPath()).AsCmd()).Run(ctx, notify)
				})
			},
		})
	}
	slog.Debug("ff command: jobs created", "workspace", cfg.WorkspaceRoot, "nbJobs", len(jobs), "duration", time.Since(start))

	opts := engine.DefaultOptions().
		WithParallel(cfg.Parallel).
		WithStopOnError(cfg.StopOnError)

	eng := engine.NewEngine(opts)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eventsCh, resultCh := eng.RunJobs(ctx, jobs)

	isInteractive := cfg.IsInteractive

	if isInteractive {
		if err := engineui.Run(eventsCh, cancel, opts.Parallel, len(jobs)); err != nil {
			slog.Error("UI error", "error", err)
		}
	} else {
		engine.ConsumeVerbose(eventsCh) // program stay in ConsumeVerbose while eventsCh is not closed
	}

	execResult := <-resultCh // pk ? pas directement resultCH ?

	for _, skipped := range skippedJobs {
		execResult.AddResult(skipped)
	}

	if !isInteractive {
		renderer := cli.NewRenderer()
		if execResult.HasErrors() {
			for _, r := range execResult.Failed() {
				renderer.RenderError(fmt.Sprintf("%s: %v", r.JobId, r.Error))
			}
		}
		renderer.RenderSuccess(fmt.Sprintf("Pulled %d repositories", execResult.SuccessCount()))
		if execResult.SkippedCount() > 0 {
			renderer.RenderWarning(fmt.Sprintf("Skipped %d repositories", execResult.SkippedCount()))
		}
	}

	if err := hooks.PostFF(cfg.WorkspaceRoot, execResult.SuccessCount()); err != nil {
		return fmt.Errorf("post-ff hook failed: %w", err)
	}

	if execResult.HasErrors() {
		return fmt.Errorf("%d repositories failed to pull", execResult.FailedCount())
	}

	return nil
}
