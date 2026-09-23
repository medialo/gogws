package check

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/hooks"
	"github.com/medialo/gogws/internal/ui/cli"
	"github.com/spf13/cobra"
	"golang.org/x/text/feature/plural"
)

var checkFlagShowKnown bool
var checkFlagShowPath bool

func NewCommand(getConfig func() *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check workspace consistency",
		Long: `Check the workspace for all repositories (known, unknown, ignored, missing).
This can be slow for large workspaces.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck(getConfig)
		},
	}

	cmd.Flags().BoolVar(&checkFlagShowKnown, "show-known", false, "list known repositories individually instead of a single summary count")
	cmd.Flags().BoolVar(&checkFlagShowPath, "show-path", false, "display the full path of each repository")

	return cmd
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

	ws, err := gws2.NewFromPath(cfg.WorkspaceRoot).Recursive(false).Load()
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}
	slog.Debug("Loaded projects", "projects", ws.Projects)

	renderer := cli.NewRenderer()

	knownByPath := make(map[string]gws2.Repository, len(ws.Projects)+len(ws.Children))
	for _, p := range ws.Projects {
		knownByPath[filepath.Clean(p.Path)] = p
	}
	for _, c := range ws.Children {
		knownByPath[filepath.Clean(c.Path)] = c
	}

	discovered, err := git.DiscoverRepositories(cfg.WorkspaceRoot, 0)
	if err != nil {
		return fmt.Errorf("failed to discover repositories: %w", err)
	}

	ignorePatterns, err := gws2.LoadIgnorePatterns(cfg.WorkspaceRoot)
	if err != nil {
		return fmt.Errorf("failed to load ignore patterns: %w", err)
	}

	// The set of repositories to check is the union of the known projects
	// (some of which may be missing on disk) and every git repository
	// discovered on disk (some of which may be unknown or ignored).
	allPaths := make(map[string]bool, len(knownByPath)+len(discovered))
	for path := range knownByPath {
		allPaths[path] = true
	}
	for _, repo := range discovered {
		allPaths[filepath.Clean(filepath.Join(cfg.WorkspaceRoot, repo.Path))] = true
	}

	sortedPaths := make([]string, 0, len(allPaths))
	for path := range allPaths {
		sortedPaths = append(sortedPaths, path)
	}
	sort.Strings(sortedPaths)

	fmt.Println(renderer.RenderInfoWithIcon(fmt.Sprintf("Checking %d repositories...", len(sortedPaths))))

	type checkEntry struct {
		label  string
		status string
	}

	var entries []checkEntry
	var unknown []string
	knownCount := 0
	missingCount := 0

	for _, path := range sortedPaths {
		label := filepath.Base(path)
		if checkFlagShowPath {
			label = fmt.Sprintf("%s (%s)", label, path)
		}

		switch {
		case gws2.IsPathIgnored(path, ignorePatterns):
			entries = append(entries, checkEntry{label, "Ignored"})
		case !isGitRepoDir(path):
			missingCount++
			entries = append(entries, checkEntry{label, "Missing"})
		case knownByPath[path] != nil:
			knownCount++
			if checkFlagShowKnown {
				entries = append(entries, checkEntry{label, "Known"})
			}
		default:
			relPath, err := filepath.Rel(cfg.WorkspaceRoot, path)
			if err != nil {
				relPath = path
			}
			unknown = append(unknown, relPath)
			entries = append(entries, checkEntry{label, "Unknown"})
		}
	}

	nameWidth := 0
	for _, e := range entries {
		if len(e.label) > nameWidth {
			nameWidth = len(e.label)
		}
	}

	for _, e := range entries {
		fmt.Println(renderer.RenderRepoStatus(e.label, e.status, nameWidth))
	}

	if !checkFlagShowKnown && knownCount > 0 {
		fmt.Println(renderer.RenderSuccess(fmt.Sprintf("%d known repositories (use --show-known to list)", knownCount)))
	}

	if missingCount > 0 {
		repo := plural.Selectf(missingCount, "%d", plural.One, "repository", plural.Other, "repositories")
		fmt.Println(renderer.RenderInfoWithIcon(fmt.Sprintf("Use 'gogws update' to clone the %d missing %s", missingCount, repo)))
	}

	if err := hooks.PostCheck(cfg.WorkspaceRoot, unknown); err != nil {
		return fmt.Errorf("post-check hook failed: %w", err)
	}

	return nil
}

func isGitRepoDir(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}
