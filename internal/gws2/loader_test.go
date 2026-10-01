package gws2

import (
	"log/slog"
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

func TestLoad_MissingProjectsFileIsNotAWarning(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigDirName, WorkspacesFileName), "sub | folder\n")
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	var logs strings.Builder
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	ws, err := NewFromPath(root).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(ws.Projects) != 0 || len(ws.Children) != 1 {
		t.Fatalf("got %d projects, %d children", len(ws.Projects), len(ws.Children))
	}
	if logs.Len() > 0 {
		t.Fatalf("unexpected warnings:\n%s", logs.String())
	}
}

func TestLoad_LegacyFilesWithoutConfigDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ProjectsFileName), "# legacy\napi | git@example.com:org/api.git\nlibs/shared | git@example.com:org/shared.git | git@example.com:up/shared.git upstream\n")
	writeFile(t, filepath.Join(root, WorkspacesFileName), ". | git@example.com:org/root.git\ntools | folder\n")
	writeFile(t, filepath.Join(root, "tools", ProjectsFileName), "cli | git@example.com:org/cli.git\n")

	if _, err := os.Stat(filepath.Join(root, ConfigDirName)); !os.IsNotExist(err) {
		t.Fatalf("test setup must not create %s", ConfigDirName)
	}

	ws, err := NewFromPath(root).Load()
	if err != nil {
		t.Fatal(err)
	}

	if len(ws.Projects) != 2 {
		t.Fatalf("root projects = %d, want 2", len(ws.Projects))
	}
	shared := ws.Projects[1]
	if shared.GetPath() != filepath.Join(root, "libs", "shared") || len(shared.Remotes) != 2 || shared.Remotes[1].Name != "upstream" {
		t.Fatalf("libs/shared = %+v, remotes %+v", shared.Entry, shared.Remotes)
	}
	if !ws.IsGitRepository() {
		t.Fatal("root '.' line from the legacy workspaces file must be read")
	}
	if len(ws.Children) != 1 || ws.Children[0].GetName() != "tools" {
		t.Fatalf("children = %+v", ws.Children)
	}
	tools := ws.Children[0]
	if len(tools.Projects) != 1 || tools.Projects[0].GetPath() != filepath.Join(root, "tools", "cli") {
		t.Fatalf("tools projects = %+v", tools.Projects)
	}
}

func TestLoad_ConfigDirWinsOverLegacyFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ProjectsFileName), "legacy | git@example.com:org/legacy.git\n")
	writeFile(t, filepath.Join(root, ConfigDirName, ProjectsFileName), "api | git@example.com:org/api.git\n")

	ws, err := NewFromPath(root).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(ws.Projects) != 1 || ws.Projects[0].GetName() != "api" {
		t.Fatalf("projects = %+v, want only api from %s", ws.Projects, ConfigDirName)
	}
}

func TestFindRoot_LegacyProjectsFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ProjectsFileName), "api | git@example.com:org/api.git\n")
	sub := filepath.Join(root, "api", "src")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	previousCache := cachedRootDir
	cachedRootDir = ""
	t.Cleanup(func() {
		_ = os.Chdir(previousDir)
		cachedRootDir = previousCache
	})
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}

	got, err := FindRoot()
	if err != nil {
		t.Fatal(err)
	}
	gotEval, _ := filepath.EvalSymlinks(got)
	wantEval, _ := filepath.EvalSymlinks(root)
	if !strings.EqualFold(gotEval, wantEval) {
		t.Fatalf("FindRoot() = %q, want %q", got, root)
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
