package cli

import (
	"fmt"
	"gogws/internal/gws2"
	"strings"

	"gogws/internal/git"
	"gogws/internal/gws"
	"gogws/internal/theme"

	"charm.land/lipgloss/v2"
)

type Renderer struct {
	theme theme.Theme
}

func NewRenderer() *Renderer {
	return &Renderer{
		theme: theme.GetTheme(),
	}
}

func (r *Renderer) RenderHeader(title string) string {
	return r.theme.HeaderBox.Render(" " + title + " ")
}

func (r *Renderer) RenderStatus(statuses []git.RepositoryStatus, workspace *gws2.Workspace, workspaceEntries []*gws2.Workspace, onlyChanges bool) string {
	var output strings.Builder
	output.WriteString(r.RenderHeader(fmt.Sprintf("GOGWS - Workspace Status - %s", workspace.Name)))
	output.WriteString("\n\n")

	if len(workspaceEntries) > 0 {
		output.WriteString(r.theme.Subtitle.Render("  Workspaces"))
		output.WriteString("\n")
		for _, ws := range workspaceEntries {
			output.WriteString(r.renderWorkspaceEntry(ws))
			output.WriteString("\n")
		}
		output.WriteString("\n")
	}

	output.WriteString(r.theme.Subtitle.Render("  Projects"))
	output.WriteString("\n")

	missing := 0
	clean := 0
	changed := 0
	errors := 0

	if len(statuses) > 0 {

		for _, status := range statuses {
			if !status.Exists {
				missing++
				if !onlyChanges {
					output.WriteString(r.renderMissingRepo(status) + "\n")
				}
				continue
			}

			if status.Error != nil {
				errors++
				output.WriteString(r.renderErrorRepo(status) + "\n")
				continue
			}

			isClean := status.Clean && !r.hasAnyBranchChanges(status.Branches)

			if isClean {
				clean++
				if !onlyChanges {
					output.WriteString(r.renderRepo(status) + "\n")
				}
				continue
			}

			changed++
			output.WriteString(r.renderRepo(status) + "\n")
		}
	} else {
		output.WriteString(r.theme.Warning.Render("  No projects found in workspace"))
	}

	output.WriteString("\n\n")
	output.WriteString(r.renderSummary(len(statuses), clean, changed, missing, errors, len(workspaceEntries)))

	return output.String()
}

func (r *Renderer) hasAnyBranchChanges(branches []git.BranchStatus) bool {
	for _, b := range branches {
		if b.Ahead > 0 || b.Behind > 0 {
			return true
		}
	}
	return false
}

func (r *Renderer) renderWorkspaceEntry(ws *gws2.Workspace) string {
	var icon, status string

	if ws.Error != nil {
		icon = r.theme.Error.Render(r.theme.Icons.Error)
		status = r.theme.Error.Render(ws.Error.Error())
	} else if !ws.Exists {
		icon = r.theme.Warning.Render(r.theme.Icons.Pending)
		status = r.theme.Subtle.Render("(not cloned)")
	} else {
		icon = r.theme.Success.Render(r.theme.Icons.Workspace)
		details := []string{fmt.Sprintf("%d projects", len(ws.Projects))}
		if len(ws.Children) > 0 {
			details = append(details, "has sub-workspaces")
		}
		status = r.theme.Subtle.Render("(" + strings.Join(details, ", ") + ")")
	}

	return fmt.Sprintf("  %s %s %s",
		icon,
		r.theme.Path.Render(ws.Path),
		status,
	)
}

func (r *Renderer) renderMissingRepo(status git.RepositoryStatus) string {
	return fmt.Sprintf("  %s %s %s",
		r.theme.Error.Render(r.theme.Icons.Error),
		r.theme.Path.Render(status.Path),
		r.theme.Subtle.Render("(missing)"),
	)
}

func (r *Renderer) renderErrorRepo(status git.RepositoryStatus) string {
	return fmt.Sprintf("  %s %s: %s",
		r.theme.Error.Render(r.theme.Icons.Error),
		r.theme.Path.Render(status.Path),
		r.theme.Error.Render(status.Error.Error()),
	)
}

func (r *Renderer) renderRepo(status git.RepositoryStatus) string {
	var output strings.Builder

	icon := r.theme.Success.Render(r.theme.Icons.Success)
	if !status.Clean || status.Uncommitted > 0 || status.Untracked > 0 {
		icon = r.theme.Warning.Render(r.theme.Icons.Warning)
	} else if r.hasAnyBranchChanges(status.Branches) {
		icon = r.theme.Warning.Render(r.theme.Icons.Warning)
	}

	header := fmt.Sprintf("  %s %s", icon, r.theme.Path.Render(status.Path))

	var workingTreeStatus []string
	if status.Uncommitted > 0 {
		workingTreeStatus = append(workingTreeStatus, r.theme.Warning.Render(fmt.Sprintf("%d uncommitted", status.Uncommitted)))
	}
	if status.Untracked > 0 {
		workingTreeStatus = append(workingTreeStatus, r.theme.Info.Render(fmt.Sprintf("%d untracked", status.Untracked)))
	}

	if len(workingTreeStatus) > 0 {
		padding := 40 - len(status.Path)
		if padding < 2 {
			padding = 2
		}
		header += strings.Repeat(" ", padding) + strings.Join(workingTreeStatus, ", ")
	}

	output.WriteString(header)
	output.WriteString("\n")

	if len(status.Branches) > 0 {
		output.WriteString(r.renderBranches(status.Branches))
	} else if status.Branch != "" {
		output.WriteString(r.renderSingleBranch(status))
	}

	return output.String()
}

func (r *Renderer) renderBranches(branches []git.BranchStatus) string {
	var output strings.Builder

	for _, branch := range branches {
		marker := "  "
		if branch.IsCurrent {
			marker = "* "
		}

		nameStyle := r.theme.Branch
		if branch.IsCurrent {
			nameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
		}

		branchName := padRight(branch.Name, 20)

		var upstream string
		if branch.Upstream != "" {
			upstream = r.theme.Subtle.Render(padRight(branch.Upstream, 25))
		} else {
			upstream = r.theme.Subtle.Render(padRight("(local)", 25))
		}

		var syncStatus string
		if branch.Upstream != "" {
			if branch.Ahead > 0 && branch.Behind > 0 {
				syncStatus = r.theme.Ahead.Render(fmt.Sprintf("↑%d", branch.Ahead)) + " " +
					r.theme.Behind.Render(fmt.Sprintf("↓%d", branch.Behind))
			} else if branch.Ahead > 0 {
				syncStatus = r.theme.Ahead.Render(fmt.Sprintf("↑%d", branch.Ahead))
			} else if branch.Behind > 0 {
				syncStatus = r.theme.Behind.Render(fmt.Sprintf("↓%d", branch.Behind))
			} else {
				syncStatus = r.theme.Success.Render("=")
			}
		}

		line := fmt.Sprintf("      %s%s %s %s",
			marker,
			nameStyle.Render(branchName),
			upstream,
			syncStatus,
		)

		output.WriteString(line)
		output.WriteString("\n")
	}

	return output.String()
}

func (r *Renderer) renderSingleBranch(status git.RepositoryStatus) string {
	var syncStatus string
	if status.HasRemote {
		if status.Ahead > 0 && status.Behind > 0 {
			syncStatus = r.theme.Ahead.Render(fmt.Sprintf("↑%d", status.Ahead)) + " " +
				r.theme.Behind.Render(fmt.Sprintf("↓%d", status.Behind))
		} else if status.Ahead > 0 {
			syncStatus = r.theme.Ahead.Render(fmt.Sprintf("↑%d", status.Ahead))
		} else if status.Behind > 0 {
			syncStatus = r.theme.Behind.Render(fmt.Sprintf("↓%d", status.Behind))
		} else {
			syncStatus = r.theme.Success.Render("=")
		}
	}

	return fmt.Sprintf("      * %s %s\n",
		r.theme.Branch.Render(padRight(status.Branch, 20)),
		syncStatus,
	)
}

func (r *Renderer) renderCleanRepo(status git.RepositoryStatus) string {
	return fmt.Sprintf("  %s %s %s",
		r.theme.Success.Render(r.theme.Icons.Success),
		r.theme.Path.Render(padRight(status.Path, 35)),
		r.theme.Branch.Render(status.Branch),
	)
}

func (r *Renderer) renderChangedRepo(status git.RepositoryStatus) string {
	var parts []string
	parts = append(parts, " ")
	parts = append(parts, r.theme.Warning.Render(r.theme.Icons.Warning))
	parts = append(parts, r.theme.Path.Render(padRight(status.Path, 35)))
	parts = append(parts, r.theme.Branch.Render(padRight(status.Branch, 15)))

	var changes []string
	if status.Uncommitted > 0 {
		changes = append(changes, r.theme.Warning.Render(fmt.Sprintf("%d uncommitted", status.Uncommitted)))
	}
	if status.Untracked > 0 {
		changes = append(changes, r.theme.Info.Render(fmt.Sprintf("%d untracked", status.Untracked)))
	}
	if status.Ahead > 0 {
		changes = append(changes, r.theme.Ahead.Render(fmt.Sprintf("↑%d", status.Ahead)))
	}
	if status.Behind > 0 {
		changes = append(changes, r.theme.Behind.Render(fmt.Sprintf("↓%d", status.Behind)))
	}

	if len(changes) > 0 {
		parts = append(parts, strings.Join(changes, " "))
	}

	return strings.Join(parts, " ")
}

func (r *Renderer) renderSummary(total, clean, changed, missing, errors, workspaces int) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Projects: %s", r.theme.Stats.Render(fmt.Sprintf("%d", total))))
	parts = append(parts, fmt.Sprintf("Clean: %s", r.theme.Success.Render(fmt.Sprintf("%d", clean))))
	parts = append(parts, fmt.Sprintf("Changed: %s", r.theme.Warning.Render(fmt.Sprintf("%d", changed))))
	parts = append(parts, fmt.Sprintf("Missing: %s", r.theme.Error.Render(fmt.Sprintf("%d", missing))))

	if errors > 0 {
		parts = append(parts, fmt.Sprintf("Errors: %s", r.theme.Error.Render(fmt.Sprintf("%d", errors))))
	}

	if workspaces > 0 {
		parts = append(parts, fmt.Sprintf("Workspaces: %s", r.theme.Info.Render(fmt.Sprintf("%d", workspaces))))
	}

	summary := strings.Join(parts, "  │  ")
	return r.theme.SummaryBox.Render(summary)
}

func (r *Renderer) RenderProjectsList(projects []gws.Project) string {
	var output strings.Builder

	output.WriteString(r.RenderHeader("Discovered Repositories"))
	output.WriteString("\n\n")

	for _, project := range projects {
		output.WriteString(fmt.Sprintf("  %s %s\n",
			r.theme.Success.Render(r.theme.Icons.Success),
			r.theme.Path.Render(project.Path),
		))
		for _, remote := range project.Remotes {
			output.WriteString(fmt.Sprintf("      %s: %s\n",
				r.theme.Remote.Render(remote.Name),
				r.theme.Subtle.Render(remote.URL),
			))
		}
	}

	output.WriteString(fmt.Sprintf("\n  %s\n",
		r.theme.Info.Render(fmt.Sprintf("Found %d repositories", len(projects))),
	))

	return output.String()
}

func (r *Renderer) RenderWorkspacesList(workspaces []*gws.Workspace) string {
	var output strings.Builder

	output.WriteString(r.RenderHeader("Workspaces"))
	output.WriteString("\n\n")

	for _, ws := range workspaces {
		output.WriteString(fmt.Sprintf("  %s %s\n",
			r.theme.Info.Render(r.theme.Icons.Workspace),
			r.theme.Path.Render(ws.Path),
		))
		output.WriteString(fmt.Sprintf("      %s: %s\n",
			r.theme.Remote.Render(ws.Remote.Name),
			r.theme.Subtle.Render(ws.Remote.URL),
		))
	}

	output.WriteString(fmt.Sprintf("\n  %s\n",
		r.theme.Info.Render(fmt.Sprintf("Found %d workspaces", len(workspaces))),
	))

	return output.String()
}

func (r *Renderer) RenderProgress(current, total int, repoPath string) string {
	percentage := float64(current) / float64(total) * 100
	return fmt.Sprintf("[%d/%d] %.0f%% - %s",
		current, total, percentage, r.theme.Path.Render(repoPath))
}

func (r *Renderer) RenderSuccess(message string) string {
	return r.theme.Success.Render(r.theme.Icons.Success + " " + message)
}

func (r *Renderer) RenderError(message string) string {
	return r.theme.Error.Render(r.theme.Icons.Error + " " + message)
}

func (r *Renderer) RenderInfo(message string) string {
	return r.theme.Info.Render(r.theme.Icons.Info + " " + message)
}

func (r *Renderer) RenderWarning(message string) string {
	return r.theme.Warning.Render(r.theme.Icons.Warning + " " + message)
}

func (r *Renderer) RenderConfigValue(key string, value interface{}, source string) string {
	sourceStyle := r.theme.Subtle
	if source == "env" {
		sourceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Italic(true)
	}

	return fmt.Sprintf("  %s: %s  %s",
		r.theme.Path.Render(key),
		r.theme.Success.Render(fmt.Sprintf("%v", value)),
		sourceStyle.Render("("+source+")"),
	)
}

func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}
