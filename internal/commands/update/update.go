package update

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/engine"
	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/hooks"
	"github.com/medialo/gogws/internal/ui/cli"
	engineui "github.com/medialo/gogws/internal/ui/engineui"

	"github.com/spf13/cobra"
)

var (
	skipProjects   bool
	skipWorkspaces bool
	recursive      bool
	prune          bool
)

func NewCommand(getConfig func() *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Clone all missing repositories and workspaces",
		Long: `Clone all repositories defined in .projects.gws and workspaces defined
in .workspaces.gws that are not yet present in the workspace.

Use --skip-projects to only clone workspaces (recursive).
Use --skip-workspaces to only clone projects.
Use --prune to remove projects and workspaces from the .gws files when
their repository can no longer be found (e.g. deleted or renamed upstream).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(getConfig)
		},
	}

	cmd.Flags().BoolVar(&skipProjects, "skip-projects", false, "skip cloning projects, only clone workspaces")
	cmd.Flags().BoolVar(&skipWorkspaces, "skip-workspaces", false, "skip cloning workspaces, only clone projects")
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Set update to recursive mode (clone all workspaces and sub-projects)")
	cmd.Flags().BoolVar(&prune, "prune", false, "Remove projects and workspaces from the .gws files when their repository cannot be found")

	return cmd
}

const maxRecursivePasses = 25

func runUpdate(getConfig func() *config.Config) error {
	cfg := getConfig()
	if cfg == nil {
		return fmt.Errorf("no workspace found (no .projects.gws file)")
	}

	if err := hooks.PreUpdate(cfg.WorkspaceRoot); err != nil {
		return fmt.Errorf("pre-update hook failed: %w", err)
	}

	renderer := cli.NewRenderer()
	var clonedProjects []string
	var prunedRepos []string

	for pass := 0; ; pass++ {
		slog.Debug(fmt.Sprintf("Running update command in workspace: %s", cfg.WorkspaceRoot), "pass", pass)
		ws, err := gws2.NewFromPath(cfg.WorkspaceRoot).Load()
		if err != nil {
			return fmt.Errorf("failed to resolve workspace: %w", err)
		}

		var didClone bool
		workspaceOwners := map[*gws2.Workspace]bool{}
		projectOwners := map[*gws2.Workspace]bool{}

		if !skipWorkspaces && len(ws.Children) > 0 {
			missingWorkspaces := ws.MissingWorkspaces()
			if recursive {
				missingWorkspaces = ws.MissingWorkspacesRecursive()
			}
			if len(missingWorkspaces) == 0 {
				fmt.Println(renderer.RenderSuccess("All workspaces are already cloned"))
			} else {
				didClone = true
				result := cloneWorkspaces(cfg.WorkspaceRoot, missingWorkspaces, cfg.Parallel, cfg.StopOnError, cfg.IsInteractive)
				if !cfg.IsInteractive {
					renderSummary(renderer, result, "Cloned workspaces")
				}

				if prune {
					removed, owners := pruneNotFound(renderer, result, ws.RemoveWorkspace)
					prunedRepos = append(prunedRepos, removed...)
					for _, o := range owners {
						workspaceOwners[o] = true
					}
				}
			}
		}

		if !skipProjects {
			missingProjects := ws.MissingProjects()
			if recursive {
				missingProjects = ws.MissingProjectsRecursive()
			}
			if len(missingProjects) == 0 {
				fmt.Println(renderer.RenderSuccess("All projects are already cloned"))
			} else {
				didClone = true
				fmt.Println(renderer.RenderInfo(fmt.Sprintf("Cloning %d missing projects...", len(missingProjects))))

				result := cloneProjects(cfg.WorkspaceRoot, missingProjects, cfg.Parallel, cfg.StopOnError, cfg.IsInteractive)
				if !cfg.IsInteractive {
					renderSummary(renderer, result, "Cloned projects")
				}

				for _, label := range result.SuccessLabels() {
					clonedProjects = append(clonedProjects, label)
				}

				if prune {
					removed, owners := pruneNotFound(renderer, result, ws.RemoveProject)
					prunedRepos = append(prunedRepos, removed...)
					for _, o := range owners {
						projectOwners[o] = true
					}
				}
			}
		}

		// Each owner is the workspace whose own .workspaces.gws/.projects.gws
		// actually held the pruned entry (root or a nested workspace); save
		// only that file, not the whole tree, so untouched config files are
		// left alone.
		for owner := range workspaceOwners {
			if err := owner.SaveWorkspace(); err != nil {
				return fmt.Errorf("failed to update %s after pruning: %w", gws2.WorkspacesFileName, err)
			}
		}
		for owner := range projectOwners {
			if err := owner.SaveProjects(); err != nil {
				return fmt.Errorf("failed to update %s after pruning: %w", gws2.ProjectsFileName, err)
			}
		}

		if !recursive || !didClone {
			break
		}
		if pass+1 >= maxRecursivePasses {
			slog.Warn("update --recursive reached the maximum number of passes, stopping", "maxPasses", maxRecursivePasses)
			break
		}
	}

	if len(prunedRepos) > 0 {
		fmt.Println(renderer.RenderWarning(fmt.Sprintf("Removed %d unreachable entries from the workspace: %s", len(prunedRepos), strings.Join(prunedRepos, ", "))))
	}

	if err := hooks.PostUpdate(cfg.WorkspaceRoot, clonedProjects); err != nil {
		return fmt.Errorf("post-update hook failed: %w", err)
	}

	return nil
}

// pruneNotFound removes, via remove, every failed job in result whose error
// indicates the remote repository no longer exists (as opposed to a
// transient network or auth failure). Returns the removed paths and the
// distinct workspace nodes whose own config file needs to be re-saved.
func pruneNotFound(renderer *cli.Renderer, result *engine.ExecutionResult, remove func(path string) *gws2.Workspace) ([]string, []*gws2.Workspace) {
	var removed []string
	var owners []*gws2.Workspace
	seen := map[*gws2.Workspace]bool{}
	for _, r := range result.Failed() {
		if !git.IsNotFoundError(r.Error) {
			continue
		}
		owner := remove(r.JobId)
		if owner == nil {
			continue
		}
		removed = append(removed, r.JobId)
		if !seen[owner] {
			seen[owner] = true
			owners = append(owners, owner)
		}
		fmt.Println(renderer.RenderWarning(fmt.Sprintf("Repository not found, removed from workspace: %s", r.JobId)))
	}
	return removed, owners
}

// todo check if engine bien placé
func cloneWorkspaces(workspaceRoot string, toClone []*gws2.Workspace, parallel int, stopOnError bool, isInteractive bool) *engine.ExecutionResult {
	if len(toClone) == 0 {
		return engine.NewNoExecutionResult()
	}

	jobs := make([]engine.Job, 0, len(toClone))

	for _, child := range toClone {

		jobs = append(jobs, engine.Job{
			JobNameId: child.GetPath(),
			Fn: func(ctx context.Context, notify engine.Notify) error {
				if child.IsGitRepository() {
					return git.CloneWorkspace(ctx, child.GetOriginRemote(), child.GetPath(), engine.WrapRunner(notify))
				}
				err := os.MkdirAll(child.GetPath(), 0755)
				if err != nil {
					return err
				}
				child.FolderExists = true
				return nil
			},
		})

	}

	return runJobs(jobs, parallel, stopOnError, isInteractive)
}

func cloneProjects(workspaceRoot string, toClone []*gws2.Project, maxParallel int, stopOnError bool, isInteractive bool) *engine.ExecutionResult {
	if len(toClone) == 0 {
		return engine.NewNoExecutionResult()
	}

	jobs := make([]engine.Job, 0, len(toClone))

	for _, p := range toClone {
		jobs = append(jobs, engine.Job{
			JobNameId: p.GetPath(),
			Fn: func(ctx context.Context, notify engine.Notify) error {
				return git.Clone(ctx, p.GetOriginRemote(), p.GetPath(), engine.WrapRunner(notify))
			},
		})
	}

	return runJobs(jobs, maxParallel, stopOnError, isInteractive)
}

func runJobs(jobs []engine.Job, maxParallel int, stopOnError bool, isInteractive bool) *engine.ExecutionResult {
	opts := engine.DefaultOptions().
		WithParallel(maxParallel).
		WithStopOnError(stopOnError)

	eng := engine.NewEngine(opts)
	events, resultCh := eng.RunJobs(context.Background(), jobs)

	if isInteractive {
		if err := engineui.Run(events, opts.Parallel, len(jobs)); err != nil {
			slog.Error("UI error", "error", err)
		}
	} else {
		engine.ConsumeVerbose(events)
	}

	return <-resultCh
}

func renderSummary(renderer *cli.Renderer, result *engine.ExecutionResult, action string) {
	if result.HasErrors() {
		for _, r := range result.Failed() {
			renderer.RenderError(fmt.Sprintf("%s: %v", r.JobId, r.Error))
		}
	}
	renderer.RenderSuccess(fmt.Sprintf("%s %d repositories", action, result.SuccessCount()))
	if result.SkippedCount() > 0 {
		renderer.RenderWarning(fmt.Sprintf("Skipped %d repositories", result.SkippedCount()))
	}
}
