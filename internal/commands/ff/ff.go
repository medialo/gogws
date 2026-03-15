package ff

import (
	"context"
	"fmt"
	"gogws/internal/engine2"
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

	renderer := cli.NewRenderer()
	output := engine.NewOutputHandler(renderer, false)

	// engine 2
	enginedos := engine2.NewEngine(engine2.DefaultOptions())
	commandsdos := make([]*engine2.EngineCommand, 0, len(ws.Projects))

	slog.Debug("ff command: creating jobs", "workspace", cfg.WorkspaceRoot, "nbProjects", len(ws.Projects))
	for _, p := range ws.Projects {
		slog.Debug("preparing job \"gows ff\"", "project", p.Path)
		repoPath := filepath.Join(cfg.WorkspaceRoot, p.Path)
		status := git.GetStatus(repoPath)

		cmd := engine2.EngineCommand(func(ctx context.Context, notify engine2.EngineCommandNotify) {
			notify(engine2.LOG, "Start job > "+repoPath)

			err := engine2.Wrap(git.Pull(repoPath).AsCmd()).Run(ctx, notify)
			if err != nil {
				return
			}
			//innerCmd := git.Pull(repoPath).AsCmd()
			//r, err := innerCmd.StdoutPipe()
			//if err != nil {
			//	return
			//}
			//err = innerCmd.Start()
			//if err != nil {
			//	return
			//}
			//scanner := bufio.NewScanner(r)
			//for scanner.Scan() {
			//	//notify(engine2.LOG, scanner.Text())
			//}
			//err = innerCmd.Wait()
			//if err != nil {
			//	notify(engine2.ERR, fmt.Sprintf("failed to pull %s: %s", repoPath, err))
			//}
			notify(engine2.OK, "End job > "+repoPath)
		})

		if !status.Exists {
			//skippedResults = append(skippedResults, engine2.Skip(cmd, "not cloned yet"))
			continue
		}

		commandsdos = append(commandsdos, &cmd)
	}
	enginedos.RunJobs(context.Background(), commandsdos)

	return nil

	// engine 1
	commands := make([]engine.RepoCommand, 0, len(ws.Projects))
	var skippedResults []engine.Result

	for _, p := range ws.Projects {
		repoPath := filepath.Join(cfg.WorkspaceRoot, p.Path)
		status := git.GetStatus(repoPath)

		cmd := engine.NewGitCommand(repoPath, p.Path, "pull", "--ff-only")
		if !status.Exists {
			skippedResults = append(skippedResults, engine.Skip(cmd, "not cloned yet"))
			continue
		}

		commands = append(commands, cmd)
	}

	result := engine.Execute(commands, engine.ExecuteOptions{
		Parallel:    cfg.Parallel,
		StopOnError: cfg.StopOnError,
	})

	for _, r := range skippedResults {
		result.AddResult(r)
	}

	output.RenderSummary(result, "Pulled")

	if err := hooks.PostFF(cfg.WorkspaceRoot, result.SuccessCount()); err != nil {
		return fmt.Errorf("post-ff hook failed: %w", err)
	}

	return nil
}
