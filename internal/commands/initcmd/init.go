package initcmd

import (
	"fmt"
	"gogws/internal/config"
	"gogws/internal/gitignore"
	"gogws/internal/gws"
	"gogws/internal/ui/cli"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	renderer                  *cli.Renderer
	ignoreGitIgnoreGeneration bool
)

func NewCommand(getConfig func() *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize workspace configuration",
		Long: `Initialize workspace configuration files.

Available subcommands:
  projects    - Discover git repositories and create projects.gws
  workspaces  - Interactively configure sub-workspaces
  gitignore   - Generate or update .gitignore for GWS

Running 'gogws init' without subcommand is equivalent to 'gogws init projects'.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			renderer = cli.NewRenderer()
			return persistentPreInitCommand(getConfig)
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			return persistentPostInitCommand(getConfig)
		},
	}

	cmd.PersistentFlags().BoolVar(&resetProjectsGwsFile, "reset", false, "reset existing projects.gws file if it exists")

	cmd.AddCommand(newProjectsCommand(getConfig))
	cmd.AddCommand(newWorkspacesCommand(getConfig))
	cmd.AddCommand(newGitignoreCommand())

	return cmd
}

//type ResetMode string
//
//const (
//	resetWorkspaceMode ResetMode = "workspaces"
//	resetProjectsMode            = "projects"
//)
//
//func resetGwsConfigFor(mode ResetMode, getConfig func() *config.Config) error {
//	slog.Debug("Resetting GWS config", "mode", mode)
//	cfg := getConfig()
//
//	gwsDir := filepath.Join(cfg.WorkspaceRoot, gws.ConfigDirName)
//	if err := os.MkdirAll(gwsDir, 0755); err != nil {
//		return fmt.Errorf("failed to create %s directory: %w", gws.ConfigDirName, err)
//	}
//
//	projectsFile := filepath.Join(gwsDir, "projects."+gws.FileExtension)
//	legacyProjectsFile := filepath.Join(cfg.WorkspaceRoot, gws.ProjectsFileName)
//
//	fileExists := false
//	if _, err := os.Stat(projectsFile); err == nil {
//		fileExists = true
//	} else if _, err := os.Stat(legacyProjectsFile); err == nil {
//		fileExists = true
//		projectsFile = legacyProjectsFile
//	}
//
//	if fileExists {
//		fmt.Println(renderer.RenderWarning(fmt.Sprintf("Removing existing %s", projectsFile)))
//		if err := os.Remove(projectsFile); err != nil {
//			return fmt.Errorf("failed to remove existing %s: %w", projectsFile, err)
//		}
//		fmt.Println(renderer.RenderSuccess(fmt.Sprintf("Removed existing %s", projectsFile)))
//		projectsFile = filepath.Join(gwsDir, "projects."+gws.FileExtension)
//	} else {
//		fmt.Println(renderer.RenderError(fmt.Sprintf("projects.%s already exists. Use --reset to reinitialize", gws.FileExtension)))
//		return nil
//	}
//}

func persistentPreInitCommand(getConfig func() *config.Config) error {
	slog.Debug("Running PersistentPreRunE", "command", "init")

	// If --reset flag is set, remove existing projects.gws file if it exists
	if resetProjectsGwsFile {
		slog.Debug("Resetting projects.gws file if it exists")
		cfg := getConfig()

		gwsDir := filepath.Join(cfg.WorkspaceRoot, gws.ConfigDirName)
		if err := os.MkdirAll(gwsDir, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", gws.ConfigDirName, err)
		}

		projectsFile := filepath.Join(gwsDir, "projects."+gws.FileExtension)
		legacyProjectsFile := filepath.Join(cfg.WorkspaceRoot, gws.ProjectsFileName)

		fileExists := false
		if _, err := os.Stat(projectsFile); err == nil {
			fileExists = true
		} else if _, err := os.Stat(legacyProjectsFile); err == nil {
			fileExists = true
			projectsFile = legacyProjectsFile
		}

		if fileExists {
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

	return nil
}

func persistentPostInitCommand(getConfig func() *config.Config) error {
	slog.Debug("Running PersistentPostRunE", "command", "init")
	if !ignoreGitIgnoreGeneration {
		cfg := getConfig()
		if err := gitignore.EnsureGWSSection(cfg.WorkspaceRoot); err != nil {
			fmt.Println(renderer.RenderWarning(fmt.Sprintf("Failed to generate .gitignore: %v", err)))
		} else {
			fmt.Println(renderer.RenderSuccess("Generated .gitignore"))
		}
	}
	return nil
}
