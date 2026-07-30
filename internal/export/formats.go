package export

import (
	"encoding/json"
	"fmt"

	"github.com/medialo/gogws/internal/view"

	"github.com/medialo/gogws/internal/git"

	"gopkg.in/yaml.v3"
)

type StatusOutput struct {
	Total        int                      `json:"total" yaml:"total"`
	Clean        int                      `json:"clean" yaml:"clean"`
	Changed      int                      `json:"changed" yaml:"changed"`
	Missing      int                      `json:"missing" yaml:"missing"`
	Errors       int                      `json:"errors" yaml:"errors"`
	Repositories []RepositoryStatusOutput `json:"repositories" yaml:"repositories"`
}

type BranchStatusOutput struct {
	Name      string `json:"name" yaml:"name"`
	IsCurrent bool   `json:"is_current" yaml:"is_current"`
	Upstream  string `json:"upstream,omitempty" yaml:"upstream,omitempty"`
	Ahead     int    `json:"ahead" yaml:"ahead"`
	Behind    int    `json:"behind" yaml:"behind"`
}

type RepositoryStatusOutput struct {
	Path        string               `json:"path" yaml:"path"`
	Exists      bool                 `json:"exists" yaml:"exists"`
	Clean       bool                 `json:"clean" yaml:"clean"`
	Branch      string               `json:"branch,omitempty" yaml:"branch,omitempty"`
	Branches    []BranchStatusOutput `json:"branches,omitempty" yaml:"branches,omitempty"`
	Ahead       int                  `json:"ahead" yaml:"ahead"`
	Behind      int                  `json:"behind" yaml:"behind"`
	Uncommitted int                  `json:"uncommitted" yaml:"uncommitted"`
	Untracked   int                  `json:"untracked" yaml:"untracked"`
	HasRemote   bool                 `json:"has_remote" yaml:"has_remote"`
	Error       string               `json:"error,omitempty" yaml:"error,omitempty"`
}

func ToJSON(statuses []*view.GitRepositoryStatusView) (string, error) {
	output := buildOutput(statuses)
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ToYAML(statuses []*view.GitRepositoryStatusView) (string, error) {
	output := buildOutput(statuses)
	data, err := yaml.Marshal(output)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func buildOutput(statusesView []*view.GitRepositoryStatusView) StatusOutput {
	output := StatusOutput{
		Total:        len(statusesView),
		Repositories: make([]RepositoryStatusOutput, len(statusesView)),
	}

	for i, statusView := range statusesView {
		status := statusView.GitStatus
		repoOutput := RepositoryStatusOutput{
			Path:        status.Path,
			Exists:      status.Exists,
			Clean:       status.Clean,
			Branch:      status.Branch,
			Ahead:       status.Ahead,
			Behind:      status.Behind,
			Uncommitted: status.Uncommitted,
			Untracked:   status.Untracked,
			HasRemote:   status.HasRemote,
		}

		if len(status.Branches) > 0 {
			repoOutput.Branches = make([]BranchStatusOutput, len(status.Branches))
			for j, branch := range status.Branches {
				repoOutput.Branches[j] = BranchStatusOutput{
					Name:      branch.Name,
					IsCurrent: branch.IsCurrent,
					Upstream:  branch.Upstream,
					Ahead:     branch.Ahead,
					Behind:    branch.Behind,
				}
			}
		}

		if status.Error != nil {
			repoOutput.Error = status.Error.Error()
			output.Errors++
		}

		if !status.Exists {
			output.Missing++
		} else if status.Clean && !hasAnyBranchChanges(status.Branches) {
			output.Clean++
		} else {
			output.Changed++
		}

		output.Repositories[i] = repoOutput
	}

	return output
}

func hasAnyBranchChanges(branches []git.BranchStatus) bool {
	for _, b := range branches {
		if b.Ahead > 0 || b.Behind > 0 {
			return true
		}
	}
	return false
}

func Format(format string, statuses ...*view.GitRepositoryStatusView) (string, error) {
	switch format {
	case "json":
		return ToJSON(statuses)
	case "yaml":
		return ToYAML(statuses)
	default:
		return "", fmt.Errorf("unsupported format: %s (supported: json, yaml)", format)
	}
}
