package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/medialo/gogws/internal/gws2"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

func (r *Renderer) RenderSearchResults(query string, matches []gws2.Repository, fullPath bool) string {
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
		t.Row(icon, kind, m.GetName(), r.RenderMatchedPath(m.GetPath(), query, fullPath))
	}
	output.WriteString(t.String())
	output.WriteString("\n")
	output.WriteString(r.RenderInfo(fmt.Sprintf("%d match(es)", len(matches))))

	return output.String()
}

func (r *Renderer) RenderMatchedPath(path, query string, fullPath bool) string {
	if query == "" {
		return r.theme.Path.Render(path)
	}

	if fullPath {
		return r.highlightSubstring(path, query)
	}

	dir, base := filepath.Split(path)
	return r.theme.Path.Render(dir) + r.highlightSubstring(base, query)
}

func (r *Renderer) highlightSubstring(s, query string) string {
	idx := strings.Index(strings.ToLower(s), strings.ToLower(query))
	if idx == -1 {
		return r.theme.Path.Render(s)
	}

	before, matched, after := s[:idx], s[idx:idx+len(query)], s[idx+len(query):]
	return r.theme.Path.Render(before) + r.theme.Match.Render(matched) + r.theme.Path.Render(after)
}
