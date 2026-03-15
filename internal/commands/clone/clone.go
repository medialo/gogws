package clone

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"gogws/internal/config"
	"gogws/internal/git"
	"gogws/internal/gws"
	"gogws/internal/hooks"
	"gogws/internal/ui/cli"

	"github.com/spf13/cobra"
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
			continue
		}

		if err := hooks.PreClone(cfg.WorkspaceRoot, repoPath); err != nil {
			fmt.Println(renderer.RenderError(fmt.Sprintf("%s: pre-clone hook failed: %v", repoPath, err)))
			continue
		}

		slog.Debug("Cloning", "path", repoPath)
		fmt.Println(renderer.RenderInfo(fmt.Sprintf("Cloning %s...", repoPath)))

		remotes := git.ToGitRemotes(project.Remotes)
		err := git.CloneWorkspace(cfg.WorkspaceRoot, project.Path, remotes)
		success := err == nil
		if err != nil {
			fmt.Println(renderer.RenderError(fmt.Sprintf("%s: %v", repoPath, err)))
		} else {
			fmt.Println(renderer.RenderSuccess(repoPath))
		}

		if hookErr := hooks.PostClone(cfg.WorkspaceRoot, repoPath, success); hookErr != nil {
			fmt.Println(renderer.RenderWarning(fmt.Sprintf("%s: post-clone hook failed: %v", repoPath, hookErr)))
		}
	}

	return nil
}
