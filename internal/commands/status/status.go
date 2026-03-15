package status

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"gogws/internal/config"
	"gogws/internal/engine"
	"gogws/internal/export"
	"gogws/internal/git"
	"gogws/internal/gws"
	"gogws/internal/ui/cli"

	"github.com/spf13/cobra"
)

func NewCommand(getConfig func() *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Aliases: []string{"st"},
		Short:   "Show the status of all repositories in the workspace",
		Long: `Display the status of all repositories defined in .projects.gws file.
Shows uncommitted changes, untracked files, and sync status with remotes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(getConfig)
		},
	}
}

func runStatus(getConfig func() *config.Config) error {
	cfg := getConfig()
	if cfg == nil {
		return fmt.Errorf("no workspace found (no .projects.gws file)")
	}

	slog.Debug("Running status command", "workspace", cfg.WorkspaceRoot)

	ws, err := gws.New(cfg.WorkspaceRoot).Load()
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	if len(ws.Projects) == 0 && len(ws.Children) == 0 {
		return fmt.Errorf("no projects or workspaces found")
	}

	slog.Debug("Found projects and workspaces", "projects", len(ws.Projects), "workspaces", len(ws.Children))

	statuses := getStatuses(cfg.WorkspaceRoot, ws.Projects, cfg.Parallel)

	if cfg.Format == "json" || cfg.Format == "yaml" {
		output, err := export.Format(statuses, cfg.Format)
		if err != nil {
			return fmt.Errorf("failed to export status: %w", err)
		}
		fmt.Println(output)
		return nil
	}

	renderer := cli.NewRenderer()
	output := renderer.RenderStatus(statuses, ws, ws.Children, cfg.OnlyChanges)
	fmt.Println(output)

	return nil
}

func getStatuses(workspaceRoot string, projects []gws.Project, parallel int) []git.RepositoryStatus {
	if len(projects) == 0 {
		return nil
	}

	var mu sync.Mutex
	statusMap := make(map[string]git.RepositoryStatus)

	jobs := make([]engine.Job, 0, len(projects))

	for _, p := range projects {
		repoPath := filepath.Join(workspaceRoot, p.Path)
		projectPath := p.Path

		jobs = append(jobs, engine.Job{
			Label: projectPath,
			Fn: func(ctx context.Context, notify engine.Notify) error {
				status := git.GetStatus(repoPath)
				status.Path = projectPath

				data, err := json.Marshal(status)
				if err != nil {
					return err
				}

				mu.Lock()
				statusMap[projectPath] = status
				mu.Unlock()

				notify(engine.EventLog, string(data))
				return nil
			},
		})
	}

	opts := engine.DefaultOptions().WithParallel(parallel)
	eng := engine.NewEngine(opts)
	events, resultCh := eng.RunJobs(context.Background(), jobs)

	for range events {
	}

	result := <-resultCh

	statuses := make([]git.RepositoryStatus, 0, len(projects))
	for _, r := range result.Results {
		if status, ok := statusMap[r.Label]; ok {
			statuses = append(statuses, status)
		} else {
			statuses = append(statuses, git.RepositoryStatus{
				Path:   r.Label,
				Exists: false,
				Error:  r.Error,
			})
		}
	}

	return statuses
}
