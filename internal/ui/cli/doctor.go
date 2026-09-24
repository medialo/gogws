package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/medialo/gogws/internal/gws2"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

// RenderDoctorRun renders one row per (workspace, check) pair, most
// severe first within each workspace, so a failure in a deeply nested
// workspace is as visible as one at the root. root is the workspace the
// checks were run from, used only to compute each row's relative label.
func (r *Renderer) RenderDoctorRun(root *gws2.Workspace, results []gws2.WorkspaceDoctorResult) string {
	var output strings.Builder

	pass := r.theme.Success.Render("[PASS]")
	fail := r.theme.Error.Render("[FAIL]")

	t := table.New().Border(lipgloss.HiddenBorder()).
		Headers("", "Workspace", "Check", "Status", "").
		StyleFunc(func(row, col int) lipgloss.Style {
			if col == 3 {
				return lipgloss.NewStyle().MarginLeft(2)
			}
			return table.DefaultStyles(0, 0)
		})

	output.WriteString(r.RenderHeader(fmt.Sprintf("GOGWS - Workspace Doctor - %s", root.Name)))
	output.WriteString("\n")
	for _, wr := range results {
		label := doctorWorkspaceLabel(root, wr.Workspace)
		for _, id := range gws2.DoctorCheckIdValues() {
			checkResult, ok := wr.Results[id]
			if !ok {
				continue
			}
			switch checkResult {
			case gws2.Passed:
				icon := r.theme.Success.Render(r.theme.Icons.Success)
				t.Row(icon, label, id.String(), pass, "")
			case gws2.Failed:
				icon := r.theme.Error.Render(r.theme.Icons.Error)
				t.Row(icon, label, id.String(), fail, "")
			case gws2.Fixed:
				icon := r.theme.Success.Render(r.theme.Icons.Success)
				t.Row(icon, label, id.String(), pass, r.theme.Warning.Render("(fixed)"))
			}
		}
	}
	output.WriteString(t.String())
	return output.String()
}

// doctorWorkspaceLabel returns ws's path relative to root, or "." for
// root itself, so nested workspaces are identifiable in the report.
func doctorWorkspaceLabel(root, ws *gws2.Workspace) string {
	if ws == root {
		return "."
	}
	rel, err := filepath.Rel(root.AbsolutePath, ws.AbsolutePath)
	if err != nil {
		return ws.Name
	}
	return rel
}
