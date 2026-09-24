package initcmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/hooks"
	"github.com/medialo/gogws/internal/providers"
	"github.com/medialo/gogws/internal/ui/cli"

	"github.com/spf13/cobra"
)

func newProviderCommand(getConfig func() *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider [github:org|gitlab:group]",
		Short: "Initialize a workspace from a git provider organization/group",
		Long: `Discover a git provider organization or group's repositories (and, for
GitLab, its subgroups) and create .gws/.projects.gws and
.gws/.workspaces.gws mirroring its structure.

Accepts a "github:<org>" or "gitlab:<group>" reference (or a full URL
after the prefix, for self-hosted instances).

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
		return fmt.Errorf("%q is not a recognized provider reference (expected a \"github:\" or \"gitlab:\" prefix)", url)
	}

	renderer := cli.NewRenderer()

	gwsDir := filepath.Join(workspaceRoot, gws2.ConfigDirName)
	exists := fileExistsAny(
		filepath.Join(gwsDir, gws2.ProjectsFileName),
		filepath.Join(workspaceRoot, gws2.ProjectsFileName),
		filepath.Join(gwsDir, gws2.WorkspacesFileName),
		filepath.Join(workspaceRoot, gws2.WorkspacesFileName),
	)

	if exists {
		if !resetProjectsGwsFile {
			fmt.Println(renderer.RenderError("workspace already has .projects.gws/.workspaces.gws. Use --reset to reinitialize"))
			return nil
		}
		if fileLocation, err := gws2.DeleteProjectsFile(workspaceRoot); fileLocation != "" {
			if err != nil {
				return fmt.Errorf("failed to remove existing %s: %w", fileLocation, err)
			}
			fmt.Println(renderer.RenderSuccess("Removed existing .projects.gws"))
		}
		if fileLocation, err := gws2.DeleteWorkspacesFile(workspaceRoot); fileLocation != "" {
			if err != nil {
				return fmt.Errorf("failed to remove existing %s: %w", fileLocation, err)
			}
			fmt.Println(renderer.RenderSuccess("Removed existing .workspaces.gws"))
		}
	}

	if err := hooks.PreInit(workspaceRoot); err != nil {
		return fmt.Errorf("pre-init hook failed: %w", err)
	}

	fmt.Println(renderer.RenderInfo(fmt.Sprintf("Discovering %s...", url)))

	group, err := provider.Discover(context.Background(), url, providers.MaxDiscoverDepth)
	if err != nil {
		return fmt.Errorf("failed to discover %s: %w", url, err)
	}

	if len(group.Projects) == 0 && len(group.Subgroups) == 0 {
		fmt.Println(renderer.RenderWarning(fmt.Sprintf("Nothing found for %s", url)))
		return nil
	}

	root := gws2.NewRootWorkspace(workspaceRoot)
	if err := providers.Materialize(root, provider, group); err != nil {
		return fmt.Errorf("failed to write workspace files: %w", err)
	}

	if len(root.Projects) > 0 {
		fmt.Println(renderer.RenderProjectsList(root.Projects))
	}
	fmt.Println(renderer.RenderSuccess(fmt.Sprintf(
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

func fileExistsAny(paths ...string) bool {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}
