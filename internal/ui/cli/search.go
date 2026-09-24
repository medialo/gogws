package cli

import (
	"fmt"
	"strings"

	"github.com/medialo/gogws/internal/gws2"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

// RenderSearchResults renders every match for query, one row per result,
// labeling each as a project or workspace.
func (r *Renderer) RenderSearchResults(query string, matches []gws2.Repository) string {
	var output strings.Builder

	output.WriteString(r.RenderHeader(fmt.Sprintf("GOGWS - Search - %q", query)))
	output.WriteString("\n")

	if len(matches) == 0 {
		output.WriteString(r.RenderWarning(fmt.Sprintf("no match for %q", query)))
		return output.String()
	}

	t := table.New().Border(lipgloss.HiddenBorder()).Headers("", "Kind", "Name", "Path")
	for _, m := range matches {
		icon := r.theme.Success.Render(r.theme.Icons.Success)
		kind := "project"
		if _, ok := m.(*gws2.Workspace); ok {
			icon = r.theme.Success.Render(r.theme.Icons.Workspace)
			kind = "workspace"
		}
		t.Row(icon, kind, m.GetName(), r.theme.Path.Render(m.GetPath()))
	}
	output.WriteString(t.String())
	output.WriteString("\n")
	output.WriteString(r.RenderInfo(fmt.Sprintf("%d match(es)", len(matches))))

	return output.String()
}
