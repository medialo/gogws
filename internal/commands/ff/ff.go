package ff

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

	ws, err := gws.New(cfg.WorkspaceRoot).Recursive(false).Load()
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	jobs := make([]engine.Job, 0, len(ws.Projects))
	var skippedJobs []engine.Result

	slog.Debug("ff command: creating jobs", "workspace", cfg.WorkspaceRoot, "nbProjects", len(ws.Projects))
	for i, p := range ws.Projects {
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
				slog.Debug("preparing job \"gows ff\"", "project", path, "index", i)
				return engine.Wrap(git.Pull(path).AsCmd()).Run(ctx, notify)
			},
		})
	}

	opts := engine.DefaultOptions().
		WithParallel(cfg.Parallel).
		WithStopOnError(cfg.StopOnError)

	eng := engine.NewEngine(opts)
	events, resultCh := eng.RunJobs(context.Background(), jobs)

	isInteractive := term.IsTerminal(int(os.Stdout.Fd()))
	//isInteractive = false // todo remove

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
