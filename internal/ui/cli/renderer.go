package cli

import (
	"fmt"
	"strings"

	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gws"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/theme"

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
	} else if !ws.FolderExists {
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

// to remove or migrate to gws 2
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

func (r *Renderer) Theme() theme.Theme {
	return r.theme
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
