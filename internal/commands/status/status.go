package status

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"sync"

	"github.com/medialo/gogws/internal/export"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/view"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/engine"
	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/ui/cli"

	"github.com/samber/lo"
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

	ws2, err := gws2.NewFromPath(cfg.WorkspaceRoot).Load()
	if err != nil {
		return err
	}

	if len(ws2.Projects) == 0 && len(ws2.Children) == 0 {
		return fmt.Errorf("no projects or workspaces found")
	}

	rootWorkspaceStatus := getStatusesView(cfg.Parallel, ws2)
	if len(rootWorkspaceStatus) > 1 {
		return fmt.Errorf("multiple workspaces found in the root workspace")
	}
	projectRepoStatus := getStatusesView(cfg.Parallel, ws2.Projects...)
	sort.Slice(projectRepoStatus, func(i, j int) bool {
		return projectRepoStatus[i].GwsRepository.Id() < projectRepoStatus[j].GwsRepository.Id()
	})
	workspaceRepoStatus := getStatusesView(cfg.Parallel, ws2.Children...)
	sort.Slice(workspaceRepoStatus, func(i, j int) bool {
		return workspaceRepoStatus[i].GwsRepository.Id() < workspaceRepoStatus[j].GwsRepository.Id()
	})

	if cfg.Format == "json" || cfg.Format == "yaml" {
		output, err := export.Format(cfg.Format, slices.Concat(rootWorkspaceStatus, workspaceRepoStatus, projectRepoStatus)...)
		if err != nil {
			return fmt.Errorf("failed to export status: %w", err)
		}
		fmt.Println(output)
		return nil
	}

	renderer := cli.NewRenderer()
	output := renderer.RenderStatus(rootWorkspaceStatus, projectRepoStatus, workspaceRepoStatus, ws2.Name, cfg.OnlyChanges)
	fmt.Println(output)

	return nil
}

func getStatusesView[T gws2.Repository](parallel int, repositories ...T) []*view.GitRepositoryStatusView {
	if len(repositories) == 0 {
		return nil
	}

	var mu sync.Mutex
	statusViewMap := make(map[string]*view.GitRepositoryStatusView, len(repositories))

	jobs := make([]engine.Job, 0, len(repositories))

	for _, repo := range repositories {
		repoPath := repo.GetPath()

		jobs = append(jobs, engine.Job{
			JobNameId: repo.GetPath(),
			Fn: func(ctx context.Context, notify engine.Notify) error {
				status := git.GetStatus(repoPath)
				status.Path = repo.GetPath()

				mu.Lock()
				statusViewMap[repo.GetPath()] = &view.GitRepositoryStatusView{
					GwsRepository: repo,
					GitStatus:     nil,
				}
				mu.Unlock()

				data, err := json.Marshal(status)
				if err != nil {
					return err
				}

				mu.Lock()
				statusViewMap[repo.GetPath()].GitStatus = status
				mu.Unlock()

				notify(engine.EventJobLog, string(data))
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

	for _, r := range result.Results {
		status, ok := statusViewMap[r.JobId]
		if ok && status.GitStatus == nil {
			status.GitStatus = &git.RepositoryStatus{
				Exists: false,
				Error:  r.Error,
				Path:   r.JobId,
			}
		}
	}

	return lo.Values(statusViewMap)
}
