package initcmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/medialo/gogws/internal/config"

	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gitignore"
	"github.com/medialo/gogws/internal/gws"
	"github.com/medialo/gogws/internal/hooks"
	"github.com/medialo/gogws/internal/ui/cli"

	"github.com/spf13/cobra"
)

var (
	resetProjectsGwsFile bool
)

func newProjectsCommand(getConfig func() *config.Config) *cobra.Command {
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
			return postRunInitProjects(getConfig)
		},
	}

	cmd.Flags().BoolVar(&resetProjectsGwsFile, "reset", false, "reset existing .projects.gws file if it exists")
	cmd.Flags().BoolVar(&ignoreGitIgnoreGeneration, "no-gitignore", true, "do not generate .gitignore file")

	return cmd
}

func preRunInitProjects(getConfig func() *config.Config) error {
	if resetProjectsGwsFile {
		cfg := getConfig()
		slog.Debug("Resetting .projects.gws config", "resetProjectsGwsFile", resetProjectsGwsFile)
		fileLocation, err := gws.DeleteProjectsFile(cfg.WorkspaceRoot)

		if fileLocation != "" {
			fmt.Println(renderer.RenderWarning("Removing projects configuration file..."))
			if err != nil {
				return fmt.Errorf("failed to remove configuration %s: %w", fileLocation, err)
			}
			fmt.Println(renderer.RenderSuccess(fmt.Sprintf("Projects configuration file removed")))
		} else {
			fmt.Println(renderer.RenderError(fmt.Sprintf("%s already exists. Use --reset to reinitialize", gws.ProjectsFileName)))
		}
	}
	return nil
}

func postRunInitProjects(getConfig func() *config.Config) error {
	if !ignoreGitIgnoreGeneration {
		workspaceRoot, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		if err := gitignore.EnsureGWSSection(workspaceRoot); err != nil {
			fmt.Println(renderer.RenderWarning(fmt.Sprintf("Failed to generate .gitignore: %v", err)))
		}
	}
	return nil
}

// to migrate to gws2
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

	gwsDir := filepath.Join(workspaceRoot, gws.ConfigDirName)
	if err := os.MkdirAll(gwsDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", gws.ConfigDirName, err)
	}

	projectsFile := filepath.Join(gwsDir, "projects."+gws.FileExtension)
	legacyProjectsFile := filepath.Join(workspaceRoot, gws.ProjectsFileName)

	fileExists := false
	if _, err := os.Stat(projectsFile); err == nil {
		fileExists = true
	} else if _, err := os.Stat(legacyProjectsFile); err == nil {
		fileExists = true
		projectsFile = legacyProjectsFile
	}

	if fileExists {
		if resetProjectsGwsFile {
			fmt.Println(renderer.RenderWarning(fmt.Sprintf("Removing existing %s", projectsFile)))
			if err := os.Remove(projectsFile); err != nil {
				return fmt.Errorf("failed to remove existing %s: %w", projectsFile, err)
			}
			fmt.Println(renderer.RenderSuccess(fmt.Sprintf("Removed existing %s", projectsFile)))
			projectsFile = filepath.Join(gwsDir, "projects."+gws.FileExtension)
		} else {
			fmt.Println(renderer.RenderError(fmt.Sprintf("projects.%s already exists. Use --reset to reinitialize", gws.FileExtension)))
			return nil
		}
	}

	fmt.Println(renderer.RenderInfo("Scanning workspace for git repositories..."))

	discovered, err := git.DiscoverRepositories(workspaceRoot, 10)
	if err != nil {
		return fmt.Errorf("failed to discover repositories: %w", err)
	}

	if len(discovered) == 0 {
		fmt.Println(renderer.RenderWarning("No git repositories found in workspace"))
		return nil
	}

	slog.Debug("Found repositories", "count", len(discovered))

	projects := make([]gws.Project, len(discovered))
	for i, d := range discovered {
		projects[i] = gws.Project{
			Path:    d.Path,
			Remotes: toGwsRemotes(d.Remotes),
		}
	}
	fmt.Println(renderer.RenderProjectsList(projects))

	file, err := os.Create(projectsFile)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", projectsFile, err)
	}
	defer file.Close()

	var projectPaths []string
	for _, project := range projects {
		projectPaths = append(projectPaths, project.Path)
		var remoteParts []string
		for _, remote := range project.Remotes {
			remoteParts = append(remoteParts, fmt.Sprintf("%s %s", remote.URL, remote.Name))
		}
		line := fmt.Sprintf("%s | %s\n", project.Path, strings.Join(remoteParts, " | "))
		if _, err := file.WriteString(line); err != nil {
			return fmt.Errorf("failed to write to %s: %w", projectsFile, err)
		}
	}

	fmt.Println(renderer.RenderSuccess(fmt.Sprintf("Created %s with %d repositories", projectsFile, len(projects))))

	if err := hooks.PostInit(workspaceRoot, projectPaths); err != nil {
		return fmt.Errorf("post-init hook failed: %w", err)
	}

	return nil
}

func toGwsRemotes(remotes []git.Remote) []gws.Remote {
	result := make([]gws.Remote, len(remotes))
	for i, r := range remotes {
		result[i] = gws.Remote{Name: r.Name, URL: r.URL}
	}
	return result
}
