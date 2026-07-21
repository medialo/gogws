package export

import (
	"encoding/json"
	"strings"
	"testing"

	"gogws/internal/git"

	"gopkg.in/yaml.v3"
)

func TestToJSON(t *testing.T) {
	statuses := []git.RepositoryStatus{
		{
			Path:        "repo1",
			Exists:      true,
			Clean:       true,
			Branch:      "main",
			Ahead:       0,
			Behind:      0,
			Uncommitted: 0,
			Untracked:   0,
			HasRemote:   true,
		},
		{
			Path:      "repo2",
			Exists:    false,
			Clean:     false,
			Branch:    "",
			HasRemote: false,
		},
	}

	output, err := ToJSON(statuses)
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var result StatusOutput
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if result.Total != 2 {
		t.Errorf("Expected total 2, got %d", result.Total)
	}

	if result.Clean != 1 {
		t.Errorf("Expected clean 1, got %d", result.Clean)
	}

	if result.Missing != 1 {
		t.Errorf("Expected missing 1, got %d", result.Missing)
	}
}

func TestToYAML(t *testing.T) {
	statuses := []git.RepositoryStatus{
		{
			Path:      "repo1",
			Exists:    true,
			Clean:     true,
			Branch:    "main",
			HasRemote: true,
		},
	}

	output, err := ToYAML(statuses)
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	var result StatusOutput
	if err := yaml.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if result.Total != 1 {
		t.Errorf("Expected total 1, got %d", result.Total)
	}

	if !strings.Contains(output, "repo1") {
		t.Error("YAML output should contain repo path")
	}
}

func TestFormat(t *testing.T) {
	statuses := []git.RepositoryStatus{
		{Path: "repo1", Exists: true, Clean: true},
	}

	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{"json format", "json", false},
		{"yaml format", "yaml", false},
		{"invalid format", "xml", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Format(tt.format, statuses)
			if (err != nil) != tt.wantErr {
				t.Errorf("Format() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
