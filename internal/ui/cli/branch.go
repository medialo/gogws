package cli

import (
	"fmt"
	"gogws/internal/git"

	"charm.land/lipgloss/v2"
	"github.com/samber/lo"
)

func (r *Renderer) renderBranches(branches []git.BranchStatus) []string {
	var branchesRendered = make([]string, 0, len(branches))
	for _, branch := range branches {
		marker := lo.Ternary(branch.IsCurrent, "* ", "  ")

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
		} else {
			syncStatus = r.theme.Warning.Render("x")
		}

		line := fmt.Sprintf("%s%s %s %s",
			marker,
			nameStyle.Render(branchName),
			upstream,
			syncStatus,
		)

		branchesRendered = append(branchesRendered, line)
	}
	return branchesRendered
}

func (r *Renderer) renderSingleBranch(status *git.RepositoryStatus) []string {
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

	return []string{fmt.Sprintf("      * %s %s\n",
		r.theme.Branch.Render(padRight(status.Branch, 20)),
		syncStatus,
	)}
}
