package fetch

import (
	"context"
	"fmt"
	"github.com/medialo/gogws/internal/gws2"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/engine"
	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/hooks"
	"github.com/medialo/gogws/internal/ui/cli"
	engineui "github.com/medialo/gogws/internal/ui/engineui"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewCommand(getConfig func() *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "fetch",
		Short: "Fetch updates from origin for all repositories",
		Long:  `Fetch updates from origin remote for all repositories in the workspace.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFetch(getConfig)
		},
	}
}

func runFetch(getConfig func() *config.Config) error {
	cfg := getConfig()
	if cfg == nil {
		return fmt.Errorf("no workspace found (no .projects.gws file)")
	}

	if err := hooks.PreFetch(cfg.WorkspaceRoot); err != nil {
		return fmt.Errorf("pre-fetch hook failed: %w", err)
	}

	slog.Debug("Running fetch command", "workspace", cfg.WorkspaceRoot)

	//ws, err := gws.New(cfg.WorkspaceRoot).Recursive(false).Load()
	ws, err := gws2.NewFromPath(cfg.WorkspaceRoot).Recursive(false).Load()
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	jobs := make([]engine.Job, 0, len(ws.Projects))
	var skippedJobs []engine.JobResult

	for _, p := range ws.Projects {
		repoPath := filepath.Join(cfg.WorkspaceRoot, p.Path)

		jobs = append(jobs, engine.Job{
			JobNameId: p.Path,
			Fn: func(ctx context.Context, notify engine.Notify) error {
				notify(engine.EventJobLog, "Checking if project is cloned...")
				status := git.GetStatus(repoPath)
				if !status.Exists {
					ctx.Done()
					return nil
				}
				notify(engine.EventJobLog, "Fetching...")
				return engine.Wrap(git.Fetch(repoPath).AsCmd()).Run(ctx, notify)
			},
		})
	}

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
		engine.ConsumeVerbose(eventsCh)
	}

	execResult := <-resultCh

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
		renderer.RenderSuccess(fmt.Sprintf("Fetched %d repositories", execResult.SuccessCount()))
		if execResult.SkippedCount() > 0 {
			renderer.RenderWarning(fmt.Sprintf("Skipped %d repositories", execResult.SkippedCount()))
		}
	}

	if err := hooks.PostFetch(cfg.WorkspaceRoot, execResult.SuccessCount()); err != nil {
		return fmt.Errorf("post-fetch hook failed: %w", err)
	}

	if execResult.HasErrors() {
		return fmt.Errorf("%d repositories failed to fetch", execResult.FailedCount())
	}

	return nil
}
