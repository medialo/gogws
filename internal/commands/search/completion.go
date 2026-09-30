package search

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/medialo/gogws/internal/utils"
)

const defaultCdAlias = "gcd"

var validAliasName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

var cdSnippetRenderers = map[utils.ShellType]func(alias string) string{
	utils.Bash:       posixCdSnippet,
	utils.Zsh:        posixCdSnippet,
	utils.Fish:       fishCdSnippet,
	utils.Powershell: powershellCdSnippet,
}

func runSearchCompletion(shellStr string, aliases []string, printFull bool) error {

	shell, ok := utils.ShellTypeString(shellStr)

	if ok != nil {
		return fmt.Errorf("unsupported --completion shell %q (want one of: %s)", shellStr, strings.Join(utils.ShellTypeStrings(), ", "))
	}

	if !printFull {
		fmt.Print(utils.SnippetScriptInit[shell](fmt.Sprintf("gogws search --completion %s --print-full-script", shell.String())))
		return nil
	}

	if len(aliases) == 0 {
		aliases = []string{defaultCdAlias}
	}
	for _, alias := range aliases {
		if !validAliasName.MatchString(alias) {
			return fmt.Errorf("invalid --alias %q: must look like a shell identifier", alias)
		}
	}

	render, _ := cdSnippetRenderers[shell]

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
