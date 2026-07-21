package cli

import (
	"fmt"
	"gogws/internal/gws2"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

func (r *Renderer) RenderDoctorRun(workspace *gws2.Workspace, results gws2.DoctorCheckResults) string {
	var output strings.Builder

	pass := r.theme.Success.Render("[PASS]")
	fail := r.theme.Error.Render("[FAIL]")

	t := table.New().Border(lipgloss.HiddenBorder()).StyleFunc(func(row, col int) lipgloss.Style {
		if col == 2 {
			return lipgloss.NewStyle().MarginLeft(10)
		}
		return table.DefaultStyles(0, 0)
	})

	output.WriteString(r.RenderHeader(fmt.Sprintf("GOGWS - Workspace Doctor - %s", workspace.Name)))
	output.WriteString("\n")
	for id, checkResult := range results {
		switch checkResult {
		case gws2.Passed:
			icon := r.theme.Success.Render(r.theme.Icons.Success)
			t.Row(icon, id.String(), pass)
		case gws2.Failed:
			icon := r.theme.Error.Render(r.theme.Icons.Error)
			t.Row(icon, id.String(), fail)
		case gws2.Fixed:
			icon := r.theme.Success.Render(r.theme.Icons.Success)
			t.Row(icon, id.String(), pass, r.theme.Warning.Render("(fixed)"))
		}
	}
	output.WriteString(t.String())
	return output.String()
}
