package search

import (
	"fmt"
	"os"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/ui/cli"

	"github.com/spf13/cobra"
)

var (
	searchFullPath bool
	searchGoToPath bool
)

func NewCommand(getConfig func() *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "search <query>",
		Aliases: []string{"find"},
		Short:   "Search projects and workspaces by name",
		Long: `Search every project and workspace in the loaded tree (root and all
nested workspaces). By default the query is matched, case-insensitively,
against each entry's final path segment (its name); use --full-path to
match against the full absolute path instead.

Use --cd (alias --go) to print the absolute path of the single match to
stdout instead of a table, for a shell "smart cd" alias, e.g.:

  gcd() { local p; p=$(gogws search --cd "$1") && cd "$p"; }
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(getConfig, args[0])
		},
	}

	cmd.Flags().BoolVar(&searchFullPath, "full-path", false, "match against the full path instead of just the name")
	cmd.Flags().BoolVar(&searchGoToPath, "cd", false, "print the absolute path to stdout when there is exactly one match")
	// pflag has no native long-name alias for a flag, only command aliases;
	// binding a second flag to the same variable is the standard way to
	// get --go to behave identically to --cd.
	cmd.Flags().BoolVar(&searchGoToPath, "go", false, "alias for --cd")

	return cmd
}

func runSearch(getConfig func() *config.Config, query string) error {
	cfg := getConfig()
	if cfg == nil {
		return fmt.Errorf("no workspace found (no .projects.gws file)")
	}

	ws, err := gws2.NewFromPath(cfg.WorkspaceRoot).RunDoctor(false).Load()
	if err != nil {
		return err
	}

	var matches []gws2.Repository
	if searchFullPath {
		matches = ws.Index().SearchByPath(query)
	} else {
		matches = ws.Index().SearchByName(query)
	}

	if searchGoToPath {
		return runSearchGoTo(matches, query)
	}

	renderer := cli.NewRenderer()
	fmt.Println(renderer.RenderSearchResults(query, matches))
	return nil
}

// runSearchGoTo implements the --cd/--go contract: stdout carries exactly
// the matched path and nothing else on success, and stays empty on
// failure (zero or multiple matches), so a shell function can safely do
// `path=$(gogws search --cd "$1") || return 1; cd "$path"`. It never goes
// through cli.Renderer, which always applies color/styling — there is no
// undecorated output mode there, so this path is a deliberate bypass.
func runSearchGoTo(matches []gws2.Repository, query string) error {
	switch len(matches) {
	case 0:
		return fmt.Errorf("no match for %q", query)
	case 1:
		fmt.Println(matches[0].GetPath())
		return nil
	default:
		fmt.Fprintf(os.Stderr, "multiple matches for %q, refine your query:\n", query)
		for _, m := range matches {
			fmt.Fprintf(os.Stderr, "  %s\n", m.GetPath())
		}
		return fmt.Errorf("%d matches for %q, expected exactly one", len(matches), query)
	}
}
