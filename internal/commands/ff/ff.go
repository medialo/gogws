package ff

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/ui/engineui"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/engine"
	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gws"
	"github.com/medialo/gogws/internal/hooks"
	"github.com/medialo/gogws/internal/ui/cli"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewCommand(getConfig func() *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "ff",
		Short: "Fast-forward pull all repositories",
		Long:  `Fast-forward pull from origin for all repositories (only if fast-forward is possible).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFF(getConfig)
		},
	}
}

func runFF(getConfig func() *config.Config) error {
	cfg := getConfig()
	if cfg == nil {
		return fmt.Errorf("no workspace found (no %s file)", gws.ProjectsFileName)
	}

	if err := hooks.PreFF(cfg.WorkspaceRoot); err != nil {
		return fmt.Errorf("pre-ff hook failed: %w", err)
	}

	slog.Debug("Running ff command", "workspace", cfg.WorkspaceRoot)

	ws, err := gws2.NewFromPath(cfg.WorkspaceRoot).Recursive(true).Load()
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	projectList := ws.FlattenProjects()
	jobs := make([]engine.Job, 0, len(projectList))
	var skippedJobs []engine.JobResult

	slog.Debug("ff command: creating jobs...", "workspace", cfg.WorkspaceRoot, "nbProjects", len(jobs))
	start := time.Now()
	for _, p := range projectList {
		jobs = append(jobs, engine.Job{
			JobNameId: p.Path,
			Fn: func(ctx context.Context, notify engine.Notify) error {
				notify(engine.EventJobLog, "Checking if project is cloned...")
				status := git.GetStatus(p.Path)

				if !status.Exists {
					ctx.Done()
					if status.Error != nil {
						return status.Error
					}
					//skippedJobs = append(skippedJobs, engine.Result{
					//	Label:      p.Path,
					//	Success:    false,
					//	Skipped:    true,
					//	SkipReason: "not cloned yet",
					//})
					return nil
				}
				notify(engine.EventJobLog, "Fast-forwarding...")

				//slog.Debug("preparing job \"gows ff\"", "project", repoPath, "index", i)
				return engine.Wrap(git.Pull(p.Path).AsCmd()).Run(ctx, notify)
			},
		})
	}
	slog.Debug("ff command: jobs created", "workspace", cfg.WorkspaceRoot, "nbJobs", len(jobs), "duration", time.Since(start))

	opts := engine.DefaultOptions().
		WithParallel(cfg.Parallel).
		WithStopOnError(cfg.StopOnError)

	eng := engine.NewEngine(opts)
	eventsCh, resultCh := eng.RunJobs(context.Background(), jobs)

	isInteractive := term.IsTerminal(int(os.Stdout.Fd()))

	if isInteractive {
		if err := engineui.Run(eventsCh, opts.Parallel, len(jobs)); err != nil {
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
