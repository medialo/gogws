package search

import (
	"errors"
	"fmt"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/ui/cli"
	"github.com/medialo/gogws/internal/ui/prompt"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

var (
	searchFullPath        bool
	searchGoToPath        bool
	searchCompletionShell string
	searchCompletionAlias []string
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

Use --cd to print the absolute path of the single match to
stdout instead of a table, for a shell "smart cd" alias, e.g.:

  gcd() { local p; p=$(gogws search --cd "$1") && cd "$p"; }

Use --completion <shell> to print that alias ready-made instead of typing
it by hand (bash, zsh, fish, powershell), to source from a shell rc file:

  gogws search --completion bash >> ~/.bashrc

Use --alias to name the generated function something other than "gcd"
(repeatable, or comma-separated):

  gogws search --completion zsh --alias gcd,gcdx >> ~/.zshrc
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if searchCompletionShell != "" {
				return runSearchCompletion(searchCompletionShell, searchCompletionAlias)
			}
			if len(searchCompletionAlias) > 0 {
				return fmt.Errorf("--alias only applies together with --completion")
			}
			if len(args) != 1 {
				return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
			}
			return runSearch(getConfig, args[0])
		},
	}

	cmd.Flags().BoolVar(&searchFullPath, "full-path", false, "match against the full path instead of just the name")
	cmd.Flags().BoolVar(&searchGoToPath, "cd", false, "print the absolute path to stdout when there is exactly one match")
	cmd.Flags().StringVar(&searchCompletionShell, "completion", "", "print a shell snippet defining a cd alias that wraps --cd (bash, zsh, fish, powershell)")
	cmd.Flags().StringSliceVar(&searchCompletionAlias, "alias", nil, `alias name(s) for the generated cd function with --completion (default "gcd")`)

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
	).WithKeyMap(prompt.KeyMap()).Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, fmt.Errorf("selection canceled for %q", query)
		}
		return nil, err
	}

	return selected, nil
}
