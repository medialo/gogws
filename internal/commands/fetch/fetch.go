package fetch

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gogws/internal/config"
	"gogws/internal/engine"
	"gogws/internal/git"
	"gogws/internal/gws"
	"gogws/internal/hooks"
	"gogws/internal/ui/cli"
	engineui "gogws/internal/ui/engineui"

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

	ws, err := gws.New(cfg.WorkspaceRoot).Recursive(false).Load()
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	jobs := make([]engine.Job, 0, len(ws.Projects))
	var skippedJobs []engine.Result

	for _, p := range ws.Projects {
		repoPath := filepath.Join(cfg.WorkspaceRoot, p.Path)
		status := git.GetStatus(repoPath)

		if !status.Exists {
			skippedJobs = append(skippedJobs, engine.Result{
				Label:      p.Path,
				Success:    false,
				Skipped:    true,
				SkipReason: "not cloned yet",
			})
			continue
		}

		path := repoPath
		jobs = append(jobs, engine.Job{
			Label: p.Path,
			Fn: func(ctx context.Context, notify engine.Notify) error {
				return engine.Wrap(git.Fetch(path).AsCmd()).Run(ctx, notify)
			},
		})
	}

	opts := engine.DefaultOptions().
		WithParallel(cfg.Parallel).
		WithStopOnError(cfg.StopOnError)

	eng := engine.NewEngine(opts)
	events, resultCh := eng.RunJobs(context.Background(), jobs)

	isInteractive := term.IsTerminal(int(os.Stdout.Fd()))

	if isInteractive {
		if err := engineui.Run(events, opts.Parallel, len(jobs)); err != nil {
			slog.Error("UI error", "error", err)
		}
	} else {
		engine.ConsumeVerbose(events)
	}

	execResult := <-resultCh

	for _, skipped := range skippedJobs {
		execResult.AddResult(skipped)
	}

	if !isInteractive {
		renderer := cli.NewRenderer()
		if execResult.HasErrors() {
			for _, r := range execResult.Failed() {
				renderer.RenderError(fmt.Sprintf("%s: %v", r.Label, r.Error))
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
