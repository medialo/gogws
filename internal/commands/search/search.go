package search

import (
	"fmt"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/ui/cli"

	"charm.land/huh/v2"
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

	renderer := cli.NewRenderer()

	if searchGoToPath {
		return runSearchGoTo(renderer, matches, query, searchFullPath)
	}

	fmt.Println(renderer.RenderSearchResults(query, matches, searchFullPath))
	return nil
}

func runSearchGoTo(renderer *cli.Renderer, matches []gws2.Repository, query string, fullPath bool) error {
	switch len(matches) {
	case 0:
		return fmt.Errorf("no match for %q", query)
	case 1:
		fmt.Println(matches[0].GetPath())
		return nil
	default:
		selected, err := promptMatchSelection(renderer, matches, query, fullPath)
		if err != nil {
			return err
		}
		fmt.Println(selected.GetPath())
		return nil
	}
}

// promptMatchSelection lets the user pick one of several ambiguous matches
// via an interactive huh select. The form renders to stderr (huh's
// default), so stdout stays reserved for the chosen path — preserving the
// --cd/--go contract for `$(gogws search --cd "$1")`.
func promptMatchSelection(renderer *cli.Renderer, matches []gws2.Repository, query string, fullPath bool) (gws2.Repository, error) {
	opts := make([]huh.Option[gws2.Repository], 0, len(matches))
	for _, m := range matches {
		label := fmt.Sprintf("%s  %s", m.GetName(), renderer.RenderMatchedPath(m.GetPath(), query, fullPath))
		opts = append(opts, huh.NewOption(label, m))
	}

	var selected gws2.Repository
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[gws2.Repository]().
				Title(fmt.Sprintf("Multiple matches for %q — pick one", query)).
				Options(opts...).
				Value(&selected),
		),
	).Run()
	if err != nil {
		return nil, err
	}

	return selected, nil
}
