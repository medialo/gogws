package clone

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/engine"
	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/gws2/loader"
	"github.com/medialo/gogws/internal/hooks"
	"github.com/medialo/gogws/internal/ui/cli"
	engineui "github.com/medialo/gogws/internal/ui/engineui"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewCommand(getConfig func() *config.RunContext) *cobra.Command {
	return &cobra.Command{
		Use:   "clone [repository...]",
		Short: "Clone specific repositories",
		Long:  `Clone one or more specific repositories by their path.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClone(getConfig, args)
		},
	}
}

func runClone(getConfig func() *config.RunContext, args []string) error {
	cfg := getConfig()
	if cfg == nil {
		return fmt.Errorf("no workspace found (no .projects.gws file)")
	}

	slog.Debug("Running clone command", "workspace", cfg.WorkspaceRoot)

	ws, err := loader.NewFromPath(cfg.WorkspaceRoot).Recursive(false).Load()
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	renderer := cli.NewRenderer()
	isInteractive := term.IsTerminal(int(os.Stdout.Fd()))

	projectHooks, err := hooks.PrepareProjectHooks(ws, hooks.ProjectHooksOptions{Command: "clone", Pre: hooks.HookPreClone, Post: hooks.HookPostClone, PerRepoWorkspace: true})
	if err != nil {
		return fmt.Errorf("failed to prepare hooks: %w", err)
	}

	jobs := make([]engine.Job, 0, len(args))
	var skipped []string

	for _, repoPath := range args {
		project := findProject(ws, repoPath)
		if project == nil {
			lipgloss.Println(renderer.RenderError(fmt.Sprintf("%s: not found in .projects.gws", repoPath)))
			continue
		}

		status := git.GetStatus(project.GetPath())
		if status.Exists {
			lipgloss.Println(renderer.RenderWarning(fmt.Sprintf("%s: already exists", repoPath)))
			skipped = append(skipped, repoPath)
			continue
		}

		jobs = append(jobs, engine.Job{
			JobNameId: repoPath,
			Fn: func(ctx context.Context, notify engine.Notify) error {
				return projectHooks.Around(ctx, project, notify, func() error {
					return git.Clone(ctx, project.GetOriginRemote(), project.GetPath(), engine.WrapRunner(notify))
				})
			},
		})
	}

	if len(jobs) == 0 {
		return nil
	}

	opts := engine.DefaultOptions().
		WithParallel(cfg.Parallel).
		WithStopOnError(cfg.StopOnError)

	eng := engine.NewEngine(opts)
	events, resultCh := eng.RunJobs(context.Background(), jobs)

	if isInteractive {
		if err := engineui.Run(events, opts.Parallel, len(jobs)); err != nil {
			slog.Error("UI error", "error", err)
		}
	} else {
		engine.ConsumeVerbose(events)
	}

	execResult := <-resultCh

	if !isInteractive {
		if execResult.HasErrors() {
			for _, r := range execResult.Failed() {
				renderer.RenderError(fmt.Sprintf("%s: %v", r.JobId, r.Error))
			}
		}
		renderer.RenderSuccess(fmt.Sprintf("Cloned %d repositories", execResult.SuccessCount()))
	}

	if execResult.HasErrors() {
		return fmt.Errorf("%d repositories failed to clone", execResult.FailedCount())
	}

	return nil
}

func findProject(ws *gws2.Workspace, arg string) *gws2.Project {
	candidates := []string{filepath.Join(ws.GetPath(), arg)}
	if abs, err := filepath.Abs(arg); err == nil {
		candidates = append(candidates, abs)
	}
	for _, p := range ws.Projects {
		for _, c := range candidates {
			if samePath(p.GetPath(), c) {
				return p
			}
		}
	}
	return nil
}

func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
