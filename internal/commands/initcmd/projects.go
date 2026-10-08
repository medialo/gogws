package initcmd

import (
	"fmt"
	"log/slog"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/gws2/loader"

	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gitignore"
	"github.com/medialo/gogws/internal/hooks"
	"github.com/medialo/gogws/internal/ui/cli"

	"github.com/spf13/cobra"
)

var (
	resetProjectsGwsFile bool
)

func newProjectsCommand(getConfig func() *config.RunContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "Discover git repositories and create projects.gws",
		Long: `Scan the workspace directory for existing git repositories and 
create a .gws/projects.gws file with all discovered repositories.

By default, also generates a .gitignore file configured for GWS workspaces.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return preRunInitProjects(getConfig)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInitProjects()
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return postRunInitProjects()
		},
	}

	cmd.Flags().BoolVar(&resetProjectsGwsFile, "reset", false, "reset existing .projects.gws file if it exists")
	cmd.Flags().BoolVar(&ignoreGitIgnoreGeneration, "no-gitignore", true, "do not generate .gitignore file")

	return cmd
}

func preRunInitProjects(getConfig func() *config.RunContext) error {
	if resetProjectsGwsFile {
		cfg := getConfig()
		if cfg == nil {
			// No workspace found yet — nothing to reset.
			return nil
		}
		slog.Debug("Resetting .projects.gws config", "resetProjectsGwsFile", resetProjectsGwsFile)
		fileLocation, err := loader.ClearProjects(cfg.WorkspaceRoot)

		if fileLocation != "" {
			lipgloss.Println(renderer.RenderWarning("Removing projects configuration..."))
			if err != nil {
				return fmt.Errorf("failed to reset projects in %s: %w", fileLocation, err)
			}
			lipgloss.Println(renderer.RenderSuccess("Projects configuration removed"))
		}
	}
	return nil
}

func postRunInitProjects() error {
	if !ignoreGitIgnoreGeneration {
		workspaceRoot, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		if err := gitignore.EnsureGWSSection(workspaceRoot); err != nil {
			lipgloss.Println(renderer.RenderWarning(fmt.Sprintf("Failed to generate .gitignore: %v", err)))
		}
	}
	return nil
}

func runInitProjects() error {
	workspaceRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := hooks.PreInit(workspaceRoot); err != nil {
		return fmt.Errorf("pre-init hook failed: %w", err)
	}

	slog.Debug("Initializing workspace", "path", workspaceRoot)

	renderer := cli.NewRenderer()

	ws, err := loader.NewFromPath(workspaceRoot).RunDoctor(false).Recursive(false).Load()
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	if len(ws.Projects) > 0 {
		if !resetProjectsGwsFile {
			lipgloss.Println(renderer.RenderError("Projects are already configured. Use --reset to reinitialize"))
			return nil
		}
		ws.Projects = ws.Projects[:0]
		ws.ReindexAll()
	}

	lipgloss.Println(renderer.RenderInfo("Scanning workspace for git repositories..."))

	discovered, err := git.DiscoverRepositories(workspaceRoot, 10)
	if err != nil {
		return fmt.Errorf("failed to discover repositories: %w", err)
	}

	if len(discovered) == 0 {
		lipgloss.Println(renderer.RenderWarning("No git repositories found in workspace"))
		return nil
	}

	slog.Debug("Found repositories", "count", len(discovered))

	var projectPaths []string
	for _, d := range discovered {
		project := gws2.NewProject(workspaceRoot, d.Path, d.Remotes)
		ws.AddProject(project)
		projectPaths = append(projectPaths, project.RelativePath)
	}
	lipgloss.Println(renderer.RenderProjectsList(ws.Projects))

	if err := ws.SaveProjects(); err != nil {
		return fmt.Errorf("failed to save projects: %w", err)
	}

	lipgloss.Println(renderer.RenderSuccess(fmt.Sprintf("Created projects configuration (%s) with %d repositories", ws.Config().Format().Name(), len(ws.Projects))))

	if err := hooks.PostInit(workspaceRoot, projectPaths); err != nil {
		return fmt.Errorf("post-init hook failed: %w", err)
	}

	return nil
}
