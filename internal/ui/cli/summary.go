package cli

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

type Summary struct {
	Total   int
	Clean   int
	Changed int
	Missing int
	Errors  int
}

func (r *Renderer) renderSummary(projects *Summary, workspaces *Summary) string {
	projectSummary := ""
	workspaceSummary := ""
	if projects != nil {
		projectSummary = r.renderSummaryLine("Projects", projects)
	}
	if workspaces != nil {
		workspaceSummary = r.renderSummaryLine("Workspaces", workspaces)
	}

	// box style
	maxWidth := max(lipgloss.Width(projectSummary), lipgloss.Width(workspaceSummary))
	right := lipgloss.NewStyle().Width(maxWidth).Align(lipgloss.Right)

	parts := []string{}
	for _, s := range []string{projectSummary, workspaceSummary} {
		if s != "" {
			parts = append(parts, right.Render(s))
		}
	}

	return r.theme.SummaryBox.Render(strings.Join(parts, "\n"))
}

func (r *Renderer) renderSummaryLine(title string, summary *Summary) string {
	var projectsRow []string
	if summary != nil {
		projectsRow = append(projectsRow, fmt.Sprintf(title+": %s", r.theme.Stats.Render(fmt.Sprintf("%d", summary.Total))))
		projectsRow = append(projectsRow, fmt.Sprintf("Clean: %s", r.theme.Success.Render(fmt.Sprintf("%d", summary.Clean))))
		projectsRow = append(projectsRow, fmt.Sprintf("Changed: %s", r.theme.Warning.Render(fmt.Sprintf("%d", summary.Changed))))
		projectsRow = append(projectsRow, fmt.Sprintf("Missing: %s", r.theme.Error.Render(fmt.Sprintf("%d", summary.Missing))))

		if summary.Errors > 0 {
			projectsRow = append(projectsRow, fmt.Sprintf("Errors: %s", r.theme.Error.Render(fmt.Sprintf("%d", summary.Errors))))
		}
		return strings.Join(projectsRow, "  │  ")
	}
	return ""
}
