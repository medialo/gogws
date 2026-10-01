package gws2

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_SelfLineIsNotAChildWorkspace(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, WorkspacesFileName), ". | git@example.com:org/root.git\nsub | folder\n")
	writeFile(t, filepath.Join(root, "sub", WorkspacesFileName), ". | folder\n")

	ws, err := NewFromPath(root).Load()
	if err != nil {
		t.Fatal(err)
	}

	all := append([]*Workspace{ws}, ws.FlattenWorkspaces()...)
	if len(all) != 2 {
		paths := make([]string, len(all))
		for i, w := range all {
			paths[i] = w.GetPath()
		}
		t.Fatalf("got %d workspaces %v, want root + sub", len(all), paths)
	}
	if ws.SelfEntry == nil || ws.SelfEntry.GetOriginRemote().URL != "git@example.com:org/root.git" {
		t.Fatalf("root self entry not kept: %+v", ws.SelfEntry)
	}
	if got, ok := ws.Index().Get(root); !ok || got != Repository(ws) {
		t.Fatalf("index entry for root = %v, want the root workspace", got)
	}
	if !ws.IsGitRepository() || ws.GetOriginRemote().URL != "git@example.com:org/root.git" {
		t.Fatalf("root must take its remote from its own '.' line, got %+v", ws.Remotes)
	}
	sub := ws.Children[0]
	if sub.IsGitRepository() {
		t.Fatalf("sub declared as folder by its parent must stay a folder, got %+v", sub.Remotes)
	}
}

func TestLoad_RootWithoutSelfLineStaysFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, WorkspacesFileName), "sub | folder\n")
	ws, err := NewFromPath(root).Load()
	if err != nil {
		t.Fatal(err)
	}
	if ws.IsGitRepository() {
		t.Fatalf("root without '.' line must not be git-backed, got %+v", ws.Remotes)
	}
}

func TestSaveWorkspace_KeepsSelfLine(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, WorkspacesFileName)
	writeFile(t, file, ". | git@example.com:org/root.git\nsub | folder\n")
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	ws, err := NewFromPath(root).Recursive(false).Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.SaveWorkspace(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 || lines[0] != ". | git@example.com:org/root.git" || lines[1] != "sub | folder" {
		t.Fatalf("saved file = %q", string(data))
	}
}
