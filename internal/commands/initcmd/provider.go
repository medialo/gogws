package initcmd

import (
	"context"
	"fmt"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/medialo/gogws/internal/gws2/loader"
	"github.com/medialo/gogws/internal/hooks"
	"github.com/medialo/gogws/internal/providers"
	"github.com/medialo/gogws/internal/ui/cli"

	"github.com/spf13/cobra"
)

func newProviderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider [github:org|gitlab:group|gitlab-graphql:group]",
		Short: "Initialize a workspace from a git provider organization/group",
		Long: `Discover a git provider organization or group's repositories (and, for
GitLab, its subgroups) and create .gws/.projects.gws and
.gws/.workspaces.gws mirroring its structure.

Accepts a "github:<org>", "gitlab:<group>" or "gitlab-graphql:<group>"
reference (or a full URL after the prefix, for self-hosted instances).
"gitlab-graphql" discovers the same GitLab structure as "gitlab" but via
GitLab's GraphQL API, which fetches deeply-nested subgroups in far fewer
requests — prefer it over "gitlab" for groups with many subgroups.

Run 'gogws update --recursive' afterward to actually clone everything.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInitProvider(args[0])
		},
	}

	cmd.Flags().BoolVar(&resetProjectsGwsFile, "reset", false, "reset existing .gws files if present")
	cmd.Flags().BoolVar(&ignoreGitIgnoreGeneration, "no-gitignore", true, "do not generate .gitignore file")

	return cmd
}

func runInitProvider(url string) error {
	workspaceRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	provider := providers.Find(url)
	if provider == nil {
		return fmt.Errorf("%q is not a recognized provider reference (expected a \"github:\", \"gitlab:\" or \"gitlab-graphql:\" prefix)", url)
	}

	renderer := cli.NewRenderer()

	if loader.IsWorkspace(workspaceRoot) {
		if !resetProjectsGwsFile {
			lipgloss.Println(renderer.RenderError("workspace is already configured. Use --reset to reinitialize"))
			return nil
		}
		if fileLocation, err := loader.ClearProjects(workspaceRoot); fileLocation != "" {
			if err != nil {
				return fmt.Errorf("failed to reset projects in %s: %w", fileLocation, err)
			}
			lipgloss.Println(renderer.RenderSuccess(fmt.Sprintf("Removed existing projects from %s", fileLocation)))
		}
		if fileLocation, err := loader.ClearWorkspaces(workspaceRoot); fileLocation != "" {
			if err != nil {
				return fmt.Errorf("failed to reset workspaces in %s: %w", fileLocation, err)
			}
			lipgloss.Println(renderer.RenderSuccess(fmt.Sprintf("Removed existing workspaces from %s", fileLocation)))
		}
	}

	if err := hooks.PreInit(workspaceRoot); err != nil {
		return fmt.Errorf("pre-init hook failed: %w", err)
	}

	lipgloss.Println(renderer.RenderInfo(fmt.Sprintf("Discovering %s...", url)))

	group, err := provider.Discover(context.Background(), url, providers.MaxDiscoverDepth)
	if err != nil {
		return fmt.Errorf("failed to discover %s: %w", url, err)
	}

	if len(group.Projects) == 0 && len(group.Subgroups) == 0 {
		lipgloss.Println(renderer.RenderWarning(fmt.Sprintf("Nothing found for %s", url)))
		return nil
	}

	root := loader.NewRootWorkspace(workspaceRoot)
	if err := providers.Materialize(root, provider, group); err != nil {
		return fmt.Errorf("failed to write workspace files: %w", err)
	}

	if len(root.Projects) > 0 {
		lipgloss.Println(renderer.RenderProjectsList(root.Projects))
	}
	lipgloss.Println(renderer.RenderSuccess(fmt.Sprintf(
		"Initialized workspace from %s: %d project(s), %d subgroup(s). Run 'gogws update --recursive' to clone.",
		url, len(root.Projects), len(root.Children))))

	var paths []string
	for _, p := range root.Projects {
		paths = append(paths, p.RelativePath)
	}
	if err := hooks.PostInit(workspaceRoot, paths); err != nil {
		return fmt.Errorf("post-init hook failed: %w", err)
	}

	return nil
}
