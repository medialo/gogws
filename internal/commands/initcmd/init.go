package initcmd

import (
	"fmt"
	"log/slog"

	"charm.land/lipgloss/v2"
	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/gitignore"
	"github.com/medialo/gogws/internal/gws2/loader"
	"github.com/medialo/gogws/internal/ui/cli"

	"github.com/spf13/cobra"
)

var (
	renderer                  *cli.Renderer
	ignoreGitIgnoreGeneration bool
)

func NewCommand(getConfig func() *config.RunContext) *cobra.Command {
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
	cmd.AddCommand(newProviderCommand())
	cmd.AddCommand(newGitignoreCommand())

	return cmd
}

func persistentPreInitCommand(getConfig func() *config.RunContext) error {
	slog.Debug("Running PersistentPreRunE", "command", "init")

	// If --reset flag is set, remove existing projects.gws file if it exists
	if resetProjectsGwsFile {
		slog.Debug("Resetting projects.gws file if it exists")
		cfg := getConfig()
		if cfg == nil {
			// No workspace found yet — nothing to reset.
			return nil
		}

		fileLocation, err := loader.ClearProjects(cfg.WorkspaceRoot)
		if err != nil {
			return fmt.Errorf("failed to reset projects in %s: %w", fileLocation, err)
		}
		if fileLocation != "" {
			lipgloss.Println(renderer.RenderSuccess(fmt.Sprintf("Removed existing projects from %s", fileLocation)))
		}
	}

	return nil
}

func persistentPostInitCommand(getConfig func() *config.RunContext) error {
	slog.Debug("Running PersistentPostRunE", "command", "init")
	if !ignoreGitIgnoreGeneration {
		cfg := getConfig()
		if cfg == nil {
			// getConfig() reflects the workspace state at process startup,
			// before RunE created anything — nothing resolvable yet, skip.
			return nil
		}
		if err := gitignore.EnsureGWSSection(cfg.WorkspaceRoot); err != nil {
			lipgloss.Println(renderer.RenderWarning(fmt.Sprintf("Failed to generate .gitignore: %v", err)))
		} else {
			lipgloss.Println(renderer.RenderSuccess("Generated .gitignore"))
		}
	}
	return nil
}
