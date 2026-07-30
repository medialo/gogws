package check

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gws"
	"github.com/medialo/gogws/internal/hooks"
	"github.com/medialo/gogws/internal/ui/cli"

	"github.com/spf13/cobra"
)

func NewCommand(getConfig func() *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check workspace consistency",
		Long: `Check the workspace for all repositories (known, unknown, ignored, missing).
This can be slow for large workspaces.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck(getConfig)
		},
	}
}

func runCheck(getConfig func() *config.Config) error {
	cfg := getConfig()
	if cfg == nil {
		return fmt.Errorf("no workspace found (no .projects.gws file)")
	}

	if err := hooks.PreCheck(cfg.WorkspaceRoot); err != nil {
		return fmt.Errorf("pre-check hook failed: %w", err)
	}

	slog.Debug("Running check command", "workspace", cfg.WorkspaceRoot)

	ws, err := gws.New(cfg.WorkspaceRoot).Recursive(false).Load()
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}
	slog.Debug("Loaded projects", "projects", ws.Projects)

	renderer := cli.NewRenderer()

	fmt.Println(renderer.RenderInfo("Checking known repositories..."))

	missing := 0
	for _, project := range ws.Projects {
		repoPath := filepath.Join(cfg.WorkspaceRoot, project.Path)
		status := git.GetStatus(repoPath)
		if !status.Exists {
			fmt.Println(renderer.RenderError(fmt.Sprintf("Missing: %s", project.Path)))
			missing++
		}
	}

	if missing == 0 {
		fmt.Println(renderer.RenderSuccess("All known repositories are present"))
	} else {
		fmt.Println(renderer.RenderWarning(fmt.Sprintf("%d repositories are missing", missing)))
	}

	fmt.Println()
	fmt.Println(renderer.RenderInfo("Scanning for unknown repositories..."))

	knownPaths := make([]string, len(ws.Projects))
	for i, p := range ws.Projects {
		knownPaths[i] = p.Path
	}

	unknown, err := git.FindUnknownRepositories(cfg.WorkspaceRoot, knownPaths)
	if err != nil {
		return fmt.Errorf("failed to check unknown repositories: %w", err)
	}

	if len(unknown) == 0 {
		fmt.Println(renderer.RenderSuccess("No unknown repositories found"))
	} else {
		fmt.Println(renderer.RenderWarning(fmt.Sprintf("Found %d unknown repositories:", len(unknown))))
		for _, path := range unknown {
			fmt.Println(renderer.RenderInfo(fmt.Sprintf("  %s", path)))
		}
	}

	if err := hooks.PostCheck(cfg.WorkspaceRoot, unknown); err != nil {
		return fmt.Errorf("post-check hook failed: %w", err)
	}

	return nil
}
