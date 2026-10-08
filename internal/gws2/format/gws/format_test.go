package gws

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gws2"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestParseRemote_DefaultOrigin(t *testing.T) {
	r, err := parseRemote("https://github.com/user/repo.git", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.URL != "https://github.com/user/repo.git" {
		t.Errorf("URL = %q, want %q", r.URL, "https://github.com/user/repo.git")
	}
	if r.Name != "origin" {
		t.Errorf("Name = %q, want %q", r.Name, "origin")
	}
}

func TestParseRemote_CustomNameAtIndex0(t *testing.T) {
	r, err := parseRemote("https://github.com/user/repo.git myremote", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name != "myremote" {
		t.Errorf("Name = %q, want %q", r.Name, "myremote")
	}
}

func TestParseRemote_DefaultUpstreamAtIndex1(t *testing.T) {
	r, err := parseRemote("https://github.com/upstream/repo.git", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name != "upstream" {
		t.Errorf("Name = %q, want %q", r.Name, "upstream")
	}
}

func TestParseRemote_CustomNameAtIndex1(t *testing.T) {
	r, err := parseRemote("https://github.com/upstream/repo.git fork", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name != "fork" {
		t.Errorf("Name = %q, want %q", r.Name, "fork")
	}
}

func TestParseRemote_Empty(t *testing.T) {
	_, err := parseRemote("", 0)
	if err == nil {
		t.Fatal("expected error for empty remote, got nil")
	}
}

func TestParseProjectLine_Valid(t *testing.T) {
	root := t.TempDir()
	url := "https://github.com/user/myproject.git"
	line := "myproject | " + url
	p, err := parseProjectLine(root, line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "myproject" {
		t.Errorf("Name = %q, want %q", p.Name, "myproject")
	}
	if len(p.Remotes) != 1 {
		t.Fatalf("len(Remotes) = %d, want 1", len(p.Remotes))
	}
	if p.Remotes[0].Name != "origin" {
		t.Errorf("Remotes[0].Name = %q, want %q", p.Remotes[0].Name, "origin")
	}
	if p.Remotes[0].URL != url {
		t.Errorf("Remotes[0].URL = %q, want %q", p.Remotes[0].URL, url)
	}
	if p.AbsolutePath != filepath.Join(root, "myproject") {
		t.Errorf("AbsolutePath = %q, want %q", p.AbsolutePath, filepath.Join(root, "myproject"))
	}
}

func TestParseProjectLine_MultipleRemotes(t *testing.T) {
	root := t.TempDir()
	line := "myproject | https://github.com/user/repo.git | https://github.com/upstream/repo.git"
	p, err := parseProjectLine(root, line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Remotes) != 2 {
		t.Fatalf("len(Remotes) = %d, want 2", len(p.Remotes))
	}
	if p.Remotes[1].Name != "upstream" {
		t.Errorf("Remotes[1].Name = %q, want %q", p.Remotes[1].Name, "upstream")
	}
}

func TestParseProjectLine_MissingRemote(t *testing.T) {
	root := t.TempDir()
	_, err := parseProjectLine(root, "myproject")
	if err == nil {
		t.Fatal("expected error for missing remote, got nil")
	}
}

func TestParseProjectLine_EmptyRemotePart(t *testing.T) {
	root := t.TempDir()
	_, err := parseProjectLine(root, "myproject | ")
	if err == nil {
		t.Fatal("expected error when remote URL is empty")
	}
}

func TestParseWorkspaceLine_Valid(t *testing.T) {
	line := "/home/user/ws | https://github.com/user/ws.git"
	ws, err := parseWorkspaceLine("", line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantPath := filepath.Join("", "/home/user/ws")
	if ws.AbsolutePath != wantPath {
		t.Errorf("AbsolutePath = %q, want %q", ws.AbsolutePath, wantPath)
	}
	if ws.Name != "ws" {
		t.Errorf("Name = %q, want %q", ws.Name, "ws")
	}
	if len(ws.Remotes) != 1 {
		t.Fatalf("len(Remotes) = %d, want 1", len(ws.Remotes))
	}
}

func TestParseWorkspaceLine_Folder(t *testing.T) {
	ws, err := parseWorkspaceLine(t.TempDir(), "sub | folder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.IsGitRepository() {
		t.Fatalf("folder workspace must not have remotes, got %+v", ws.Remotes)
	}
}

func TestParseWorkspaceLine_MissingURL(t *testing.T) {
	_, err := parseWorkspaceLine("", "/home/user/ws | ")
	if err == nil {
		t.Fatal("expected error for empty remote URL")
	}
}

func TestParseWorkspaceLine_InvalidFormat(t *testing.T) {
	_, err := parseWorkspaceLine("", "/home/user/ws")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestParseWorkspaceLine_EmptyPath(t *testing.T) {
	_, err := parseWorkspaceLine("", " | https://github.com/user/ws.git")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestParseProjectsFile_WithCommentsAndBlanks(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ProjectsFileName)
	writeFile(t, path, "# header\n\nprojectA | https://github.com/u/a.git\nprojectB | https://github.com/u/b.git # inline comment\n")

	projects, err := parseProjectsFile(root, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("len = %d, want 2", len(projects))
	}
}

func TestParseWorkspacesFile_FileNotExist(t *testing.T) {
	root := t.TempDir()
	workspaces, err := parseWorkspacesFile(root, filepath.Join(root, WorkspacesFileName))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(workspaces) != 0 {
		t.Errorf("len = %d, want 0", len(workspaces))
	}
}

func TestParseWorkspacesFile_Valid(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, WorkspacesFileName)
	writeFile(t, path, "# comment\n/ws/alpha | https://github.com/u/alpha.git\n/ws/beta | https://github.com/u/beta.git\n")

	workspaces, err := parseWorkspacesFile(root, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(workspaces) != 2 {
		t.Fatalf("len = %d, want 2", len(workspaces))
	}
	if workspaces[0].Name != "alpha" {
		t.Errorf("Name = %q, want %q", workspaces[0].Name, "alpha")
	}
}

func TestFormat_CanRead(t *testing.T) {
	cases := map[string]string{
		"legacy projects":       ProjectsFileName,
		"legacy workspaces":     WorkspacesFileName,
		"config dir projects":   filepath.Join(gws2.ConfigDirName, ProjectsFileNameInDir),
		"config dir workspaces": filepath.Join(gws2.ConfigDirName, WorkspacesFileNameInDir),
	}
	for name, rel := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, rel), "api | git@example.com:org/api.git\n")
			if !canRead(root) {
				t.Fatalf("CanRead(%s) = false", rel)
			}
		})
	}
	if canRead(t.TempDir()) {
		t.Fatal("CanRead on empty dir = true")
	}
}

func TestFormat_ReadExtractsSelfLine(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, WorkspacesFileName), ". | git@example.com:org/root.git\nsub | folder\n")

	node, err := open(root).Read()
	if err != nil {
		t.Fatal(err)
	}
	if node.SelfEntry == nil || node.SelfEntry.GetOriginRemote().URL != "git@example.com:org/root.git" {
		t.Fatalf("self entry = %+v", node.SelfEntry)
	}
	if len(node.Children) != 1 || node.Children[0].GetName() != "sub" {
		t.Fatalf("children = %+v", node.Children)
	}
}

func TestFormat_WriteRoundTrip(t *testing.T) {
	root := t.TempDir()
	ws := gws2.NewRootWorkspace(root)
	ws.AddProject(gws2.NewProject(root, "api", []*git.Remote{{Name: "origin", URL: "git@example.com:org/api.git"}, {Name: "upstream", URL: "git@example.com:up/api.git"}}))
	ws.AddWorkspace(gws2.NewFolderWorkspace(root, "tools"))

	if err := open(root).WriteProjects(ws); err != nil {
		t.Fatal(err)
	}
	if err := open(root).WriteWorkspaces(ws); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(root, gws2.ConfigDirName, ProjectsFileNameInDir))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "api | git@example.com:org/api.git | git@example.com:up/api.git upstream" {
		t.Fatalf("projects file = %q", string(data))
	}

	node, err := open(root).Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(node.Projects) != 1 || len(node.Projects[0].Remotes) != 2 || len(node.Children) != 1 || node.Children[0].IsGitRepository() {
		t.Fatalf("round trip = projects %+v, children %+v", node.Projects, node.Children)
	}
}

func TestFormat_ClearProjects(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ProjectsFileName)
	writeFile(t, path, "api | git@example.com:org/api.git\n")

	removed, err := open(root).ClearProjects()
	if err != nil {
		t.Fatal(err)
	}
	if removed != path {
		t.Fatalf("removed = %q, want %q", removed, path)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("projects file still exists")
	}
	if removed, _ := open(root).ClearProjects(); removed != "" {
		t.Fatalf("second clear removed %q", removed)
	}
}

func TestConfig_ReusesHandleAcrossWriteClearWrite(t *testing.T) {
	root := t.TempDir()
	config := (Format{}).New(root)
	ws := gws2.NewRootWorkspace(root)
	ws.AddProject(gws2.NewProject(root, "api", []*git.Remote{{Name: "origin", URL: "git@example.com:org/api.git"}}))

	if err := config.WriteProjects(ws); err != nil {
		t.Fatal(err)
	}
	node, err := config.Read()
	if err != nil || len(node.Projects) != 1 {
		t.Fatalf("read after write: %v, %+v", err, node)
	}
	if removed, err := config.ClearProjects(); err != nil || removed != config.Path() {
		t.Fatalf("ClearProjects = %q, %v", removed, err)
	}
	if removed, _ := config.ClearProjects(); removed != "" {
		t.Fatalf("second clear removed %q", removed)
	}
	if err := config.WriteProjects(ws); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(config.Path()); err != nil {
		t.Fatalf("file should be recreated at %s: %v", config.Path(), err)
	}
}

func canRead(dir string) bool {
	_, ok := (Format{}).Detect(gws2.ScanConfigFiles(dir))
	return ok
}

func open(dir string) gws2.WorkspaceConfig {
	if config, ok := (Format{}).Detect(gws2.ScanConfigFiles(dir)); ok {
		return config
	}
	return (Format{}).New(dir)
}
