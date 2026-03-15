package initcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"gogws/internal/gitignore"
	"gogws/internal/ui/cli"

	"github.com/spf13/cobra"
)

var (
	forceGitignore  bool
	removeGitignore bool
)

func newGitignoreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gitignore",
		Short: "Generate or update .gitignore for GWS",
		Long: `Generate or update the .gitignore file with GWS-specific rules.

The GWS section in .gitignore:
- Ignores all files and directories (sub-projects and workspaces)
- Tracks GWS configuration files (.gws/ directory and *.gws files)
- Tracks README.md and .gitignore itself

If a .gitignore already exists, the GWS section will be added or updated
without affecting other rules.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInitGitignore()
		},
	}

	cmd.Flags().BoolVar(&forceGitignore, "force", false, "force update even if GWS section from .gitingore already exists")
	cmd.Flags().BoolVar(&removeGitignore, "remove", false, "remove GWS section from .gitignore")

	return cmd
}

func runInitGitignore() error {
	workspaceRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	renderer := cli.NewRenderer()
	gitignorePath := filepath.Join(workspaceRoot, ".gitignore")

	if removeGitignore {
		hasSection, err := gitignore.HasGWSSection(gitignorePath)
		if err != nil {
			return fmt.Errorf("failed to check .gitignore: %w", err)
		}

		if !hasSection {
			fmt.Println(renderer.RenderWarning("No GWS section found in .gitignore"))
			return nil
		}

		if err := gitignore.RemoveGWSSection(gitignorePath); err != nil {
			return fmt.Errorf("failed to remove GWS section: %w", err)
		}

		fmt.Println(renderer.RenderSuccess("Removed GWS section from .gitignore"))
		return nil
	}

	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		if err := gitignore.CreateGitignore(workspaceRoot); err != nil {
			return fmt.Errorf("failed to create .gitignore: %w", err)
		}
		fmt.Println(renderer.RenderSuccess("Created .gitignore with GWS configuration"))
		return nil
	}

	hasSection, err := gitignore.HasGWSSection(gitignorePath)
	if err != nil {
		return fmt.Errorf("failed to check .gitignore: %w", err)
	}

	if hasSection && !forceGitignore {
		fmt.Println(renderer.RenderInfo("GWS section already exists in .gitignore"))
		fmt.Println(renderer.RenderInfo("Use --force to update it"))
		return nil
	}

	if err := gitignore.EnsureGWSSection(workspaceRoot); err != nil {
		return fmt.Errorf("failed to update .gitignore: %w", err)
	}

	if hasSection {
		fmt.Println(renderer.RenderSuccess("Updated GWS section in .gitignore"))
	} else {
		fmt.Println(renderer.RenderSuccess("Added GWS section to .gitignore"))
	}

	return nil
}
