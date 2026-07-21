package cli

import (
	"fmt"
	"gogws/internal/gws2"
	"gogws/internal/view"
	"strings"

	"charm.land/lipgloss/v2/tree"
	"github.com/samber/lo"
)

func (r *Renderer) RenderStatus(rootWorkspaceStatus, projectRepoStatus, childrenRepoStatus []*view.GitRepositoryStatusView, workspaceRootName string, onlyChanges bool) string {
	var output strings.Builder

	output.WriteString(r.RenderHeader(fmt.Sprintf("GOGWS - Workspace Status - %s", workspaceRootName)))

	output.WriteString("\n\n")

	// workspace status
	str, childrenSummary := r.renderStatusRepositoriesWorkspace(rootWorkspaceStatus, childrenRepoStatus, gws2.RepositoryTypeWorkspace, onlyChanges)
	output.WriteString(r.theme.MarginLeft.Render(str))

	output.WriteString("\n\n")

	// projects status
	str, projectSummary := r.renderStatusRepositoriesProject(projectRepoStatus, gws2.RepositoryTypeProject, onlyChanges)
	output.WriteString(r.theme.MarginLeft.Render(str))

	output.WriteString("\n\n")
	// summary box
	output.WriteString(r.renderSummary(projectSummary, childrenSummary))

	if onlyChanges {
		output.WriteString("\n")
		output.WriteString(" " + r.theme.SubtitleI.Render("Only showing changes"))
	}

	return output.String()
}

func (r *Renderer) renderStatusRepositoriesWorkspace(rootWorkspaceStatus, projectRepoStatus []*view.GitRepositoryStatusView, repoType gws2.RepositoryType, onlyChanges bool) (string, *Summary) {
	if len(projectRepoStatus) == 0 {
		return "No workspace found in workspace", nil
	}
	var output strings.Builder

	summary := &Summary{
		Total: len(projectRepoStatus),
	}

	title := repoType.String()
	if len(projectRepoStatus) > 1 {
		title += "s"
	}
	output.WriteString(r.theme.Subtitle.Render(title))
	output.WriteString("\n")

	repositoryTitleRendered, branchesRendered := r.renderRepository(rootWorkspaceStatus[0], summary, onlyChanges)

	hasMore := len(projectRepoStatus) > 1
	branchPrefix := lo.Ternary(hasMore, "│", "")

	parts := []string{repositoryTitleRendered}
	if len(branchesRendered) > 0 {
		rendered := lo.Map(branchesRendered, func(item string, _ int) string {
			return branchPrefix + r.theme.MarginLeftFunc(2).Render(item)
		})
		parts = append(parts, strings.Join(rendered, "\n"))
	}

	t := tree.
		Root(strings.Join(parts, "\n")).
		Enumerator(tree.RoundedEnumerator)

	for _, statusView := range projectRepoStatus {
		repositoryTitleRendered, branchesRendered := r.renderRepository(statusView, summary, onlyChanges)

		parts := []string{repositoryTitleRendered}
		if len(branchesRendered) > 0 {
			rendered := lo.Map(branchesRendered, func(item string, _ int) string {
				return r.theme.MarginLeftFunc(2).Render(item)
			})
			parts = append(parts, strings.Join(rendered, "\n"))
		}

		t.Child(strings.Join(parts, "\n"))
	}

	output.WriteString(t.String())
	return output.String(), summary
}

func (r *Renderer) renderStatusRepositoriesProject(projectRepoStatus []*view.GitRepositoryStatusView, repoType gws2.RepositoryType, onlyChanges bool) (string, *Summary) {
	if len(projectRepoStatus) == 0 {
		return "No projects found in workspace", nil
	}
	var output strings.Builder

	summary := &Summary{
		Total: len(projectRepoStatus),
	}

	title := repoType.String()
	if len(projectRepoStatus) > 1 {
		title += "s"
	}
	output.WriteString(r.theme.Subtitle.Render(title))
	output.WriteString("\n")
	for _, statusView := range projectRepoStatus {
		repositoryTitleRendered, branchesRendered := r.renderRepository(statusView, summary, onlyChanges)

		parts := []string{repositoryTitleRendered}
		if len(branchesRendered) > 0 {
			parts = append(parts, r.theme.MarginLeft.Render(strings.Join(branchesRendered, "\n")))
		}

		output.WriteString(strings.Join(parts, "\n") + "\n")
	}

	return output.String(), summary
}
