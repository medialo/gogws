package clone

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
		Use:   "clone [repository...]",
		Short: "Clone specific repositories",
		Long:  `Clone one or more specific repositories by their path.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClone(getConfig, args)
		},
	}
}

func runClone(getConfig func() *config.Config, args []string) error {
	cfg := getConfig()
	if cfg == nil {
		return fmt.Errorf("no workspace found (no .projects.gws file)")
	}

	slog.Debug("Running clone command", "workspace", cfg.WorkspaceRoot)

	ws, err := gws.New(cfg.WorkspaceRoot).Recursive(false).Load()
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	projectMap := make(map[string]gws.Project)
	for _, project := range ws.Projects {
		projectMap[project.Path] = project
	}

	renderer := cli.NewRenderer()
	isInteractive := term.IsTerminal(int(os.Stdout.Fd()))

	jobs := make([]engine.Job, 0, len(args))
	var skipped []string

	for _, repoPath := range args {
		project, exists := projectMap[repoPath]
		if !exists {
			fmt.Println(renderer.RenderError(fmt.Sprintf("%s: not found in .projects.gws", repoPath)))
			continue
		}

		fullPath := filepath.Join(cfg.WorkspaceRoot, project.Path)
		status := git.GetStatus(fullPath)
		if status.Exists {
			fmt.Println(renderer.RenderWarning(fmt.Sprintf("%s: already exists", repoPath)))
			skipped = append(skipped, repoPath)
			continue
		}

		if err := hooks.PreClone(cfg.WorkspaceRoot, repoPath); err != nil {
			fmt.Println(renderer.RenderError(fmt.Sprintf("%s: pre-clone hook failed: %v", repoPath, err)))
			continue
		}

		remotes := git.ToGitRemotes(project.Remotes)
		wsRoot := cfg.WorkspaceRoot
		projectPath := project.Path

		jobs = append(jobs, engine.Job{
			Label: repoPath,
			Fn: func(ctx context.Context, notify engine.Notify) error {
				return git.CloneWorkspace(ctx, wsRoot, projectPath, remotes, engine.WrapRunner(notify))
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

	for _, label := range execResult.SuccessLabels() {
		if hookErr := hooks.PostClone(cfg.WorkspaceRoot, label, true); hookErr != nil {
			fmt.Println(renderer.RenderWarning(fmt.Sprintf("%s: post-clone hook failed: %v", label, hookErr)))
		}
	}

	for _, label := range execResult.FailedLabels() {
		if hookErr := hooks.PostClone(cfg.WorkspaceRoot, label, false); hookErr != nil {
			fmt.Println(renderer.RenderWarning(fmt.Sprintf("%s: post-clone hook failed: %v", label, hookErr)))
		}
	}

	if !isInteractive {
		if execResult.HasErrors() {
			for _, r := range execResult.Failed() {
				renderer.RenderError(fmt.Sprintf("%s: %v", r.Label, r.Error))
			}
		}
		renderer.RenderSuccess(fmt.Sprintf("Cloned %d repositories", execResult.SuccessCount()))
	}

	return nil
}
