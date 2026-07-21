package cli

import (
	"fmt"
	"gogws/internal/git"
	"gogws/internal/view"
	"strings"
)

// renderRepository return the repository name rendered and a list of branches
func (r *Renderer) renderRepository(repositoryStatusView *view.GitRepositoryStatusView, summary *Summary, onlyChanges bool) (string, []string) {
	status := repositoryStatusView.GitStatus
	if !status.Exists {
		if !repositoryStatusView.GwsRepository.IsGitRepository() {
			summary.Clean++
			return r.renderFolderRepo(status), []string{}
		}
		summary.Missing++
		if !onlyChanges {
			return r.renderMissingRepo(status), []string{}
		}
	}

	if status.Error != nil {
		summary.Errors++
		return r.renderErrorRepo(status), []string{}
	}

	isClean := status.Clean && !r.hasAnyBranchChanges(status.Branches)

	if isClean {
		summary.Clean++
		if !onlyChanges {
			if status.Oid == "(initial)" {
				return r.renderEmptyRepo(status), []string{}
			}
			return r.renderRepo(status)
		}
	}

	summary.Changed++
	return r.renderRepo(status)
}

func (r *Renderer) renderMissingRepo(status *git.RepositoryStatus) string {
	return fmt.Sprintf("%s %s %s",
		r.theme.Warning.Render(r.theme.Icons.Pending),
		r.theme.Path.Render(status.Name()),
		r.theme.Subtle.Render("(not cloned)"),
	)
}

func (r *Renderer) renderErrorRepo(status *git.RepositoryStatus) string {
	return fmt.Sprintf("%s %s: %s",
		r.theme.Error.Render(r.theme.Icons.Error),
		r.theme.Path.Render(status.Name()),
		r.theme.Error.Render(status.Error.Error()),
	)
}

func (r *Renderer) renderEmptyRepo(status *git.RepositoryStatus) string {
	return fmt.Sprintf("%s %s: (%s)",
		r.theme.Warning.Render(r.theme.Icons.Pending),
		r.theme.Path.Render(status.Name()),
		r.theme.Warning.Render("empty"),
	)
}

func (r *Renderer) renderFolderRepo(status *git.RepositoryStatus) string {
	icon := r.theme.Emoji.Folder
	return fmt.Sprintf("%s %s (%s)", icon, r.theme.Path.Render(status.Name()), r.theme.Warning.Render("folder"))
}

func (r *Renderer) renderRepo(status *git.RepositoryStatus) (string, []string) {
	var output strings.Builder

	icon := r.theme.Success.Render(r.theme.Icons.Success)
	if !status.Clean || status.Uncommitted > 0 || status.Untracked > 0 {
		icon = r.theme.Warning.Render(r.theme.Icons.Warning)
	} else if r.hasAnyBranchChanges(status.Branches) {
		icon = r.theme.Warning.Render(r.theme.Icons.Warning)
	}

	header := fmt.Sprintf("%s %s", icon, r.theme.Path.Render(status.Name()))

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
	//output.WriteString("\n")

	var branchesRendered []string
	if len(status.Branches) > 0 {
		branchesRendered = r.renderBranches(status.Branches)
	} else if status.Branch != "" {
		branchesRendered = r.renderSingleBranch(status)
	}

	return output.String(), branchesRendered
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
