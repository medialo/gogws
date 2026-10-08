package status

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"charm.land/lipgloss/v2"
	"github.com/medialo/gogws/internal/export"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/gws2/loader"
	"github.com/medialo/gogws/internal/view"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/engine"
	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/ui/cli"

	"github.com/spf13/cobra"
)

func NewCommand(getConfig func() *config.RunContext) *cobra.Command {
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

func runStatus(getConfig func() *config.RunContext) error {
	cfg := getConfig()
	if cfg == nil {
		return fmt.Errorf("no workspace found (no .projects.gws file)")
	}

	slog.Debug("Running status command", "workspace", cfg.WorkspaceRoot)

	ws2, err := loader.NewFromPath(cfg.WorkspaceRoot).Recursive(false).Load()
	if err != nil {
		return err
	}

	if len(ws2.Projects) == 0 && len(ws2.Children) == 0 {
		return fmt.Errorf("no projects or workspaces found")
	}

	statuses := getStatusesView(cfg.Parallel, ws2)
	rootWorkspaceStatus := statuses[:1]
	projectRepoStatus := statuses[1 : 1+len(ws2.Projects)]
	workspaceRepoStatus := statuses[1+len(ws2.Projects):]

	if cfg.Format == "json" || cfg.Format == "yaml" {
		output, err := export.Format(cfg.Format, slices.Concat(rootWorkspaceStatus, workspaceRepoStatus, projectRepoStatus)...)
		if err != nil {
			return fmt.Errorf("failed to export status: %w", err)
		}
		lipgloss.Println(output)
		return nil
	}

	renderer := cli.NewRenderer()
	output := renderer.RenderStatus(rootWorkspaceStatus, projectRepoStatus, workspaceRepoStatus, ws2.Name, cfg.OnlyChanges)
	lipgloss.Println(output)

	return nil
}

func getStatusesView(parallel int, ws *gws2.Workspace) []*view.GitRepositoryStatusView {
	repositories := make([]gws2.Repository, 0, 1+len(ws.Projects)+len(ws.Children))
	repositories = append(repositories, ws)
	for _, p := range ws.Projects {
		repositories = append(repositories, p)
	}
	for _, c := range ws.Children {
		repositories = append(repositories, c)
	}

	views := make([]*view.GitRepositoryStatusView, len(repositories))
	jobs := make([]engine.Job, 0, len(repositories))

	for i, repo := range repositories {
		repoPath := repo.GetPath()
		views[i] = &view.GitRepositoryStatusView{GwsRepository: repo}
		jobs = append(jobs, engine.Job{
			JobNameId: repoPath,
			Fn: func(ctx context.Context, notify engine.Notify) error {
				views[i].GitStatus = git.GetStatus(repoPath)
				return nil
			},
		})
	}

	opts := engine.DefaultOptions().WithParallel(parallel)
	eng := engine.NewEngine(opts)
	events, resultCh := eng.RunJobs(context.Background(), jobs)

	for range events {
	}
	<-resultCh

	for _, v := range views {
		if v.GitStatus == nil {
			v.GitStatus = &git.RepositoryStatus{Exists: false, Path: v.GwsRepository.GetPath()}
		}
	}

	return views
}
