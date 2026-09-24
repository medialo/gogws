package search

import (
	"fmt"
	"regexp"
	"strings"
)

const defaultCdAlias = "gcd"

var validAliasName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

var cdSnippetRenderers = map[string]func(alias string) string{
	"bash":       posixCdSnippet,
	"zsh":        posixCdSnippet,
	"fish":       fishCdSnippet,
	"powershell": powershellCdSnippet,
}

func supportedCompletionShells() []string {
	shells := make([]string, 0, len(cdSnippetRenderers))
	for shell := range cdSnippetRenderers {
		shells = append(shells, shell)
	}
	return shells
}

func runSearchCompletion(shell string, aliases []string) error {
	if len(aliases) == 0 {
		aliases = []string{defaultCdAlias}
	}
	for _, alias := range aliases {
		if !validAliasName.MatchString(alias) {
			return fmt.Errorf("invalid --alias %q: must look like a shell identifier", alias)
		}
	}

	render, ok := cdSnippetRenderers[shell]
	if !ok {
		return fmt.Errorf("unsupported --completion shell %q (want one of: %s)", shell, strings.Join(supportedCompletionShells(), ", "))
	}

	var out strings.Builder
	for _, alias := range aliases {
		out.WriteString(render(alias))
	}
	fmt.Print(out.String())

	return nil
}

func posixCdSnippet(alias string) string {
	return fmt.Sprintf(`%s() {
	local p
	p=$(gogws search --cd "$1") && cd "$p"
}
`, alias)
}

func fishCdSnippet(alias string) string {
	return fmt.Sprintf(`function %s
	set -l p (gogws search --cd $argv[1])
	and cd $p
end
`, alias)
}

func powershellCdSnippet(alias string) string {
	return fmt.Sprintf(`function %s {
	param([string]$Query)
	$p = gogws search --cd $Query
	if ($LASTEXITCODE -eq 0) { Set-Location $p }
}
`, alias)
}
