package gws2

import (
	"os"
	"path/filepath"
	"testing"
)

// ─── helpers ────────────────────────────────────────────────────────────────

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// ─── parseRemote ────────────────────────────────────────────────────────────

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

// ─── parseProjectLine ────────────────────────────────────────────────────────

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
	if p.Path != filepath.Join(root, "myproject") {
		t.Errorf("Path = %q, want %q", p.Path, filepath.Join(root, "myproject"))
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

// ─── parseWorkspaceLine ──────────────────────────────────────────────────────

func TestParseWorkspaceLine_Valid(t *testing.T) {
	line := "/home/user/ws | https://github.com/user/ws.git"
	ws, err := parseWorkspaceLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.Path != "/home/user/ws" {
		t.Errorf("Path = %q, want %q", ws.Path, "/home/user/ws")
	}
	if ws.Name != "ws" {
		t.Errorf("Name = %q, want %q", ws.Name, "ws")
	}
	if len(ws.Remotes) != 1 {
		t.Fatalf("len(Remotes) = %d, want 1", len(ws.Remotes))
	}
}

func TestParseWorkspaceLine_MissingURL(t *testing.T) {
	_, err := parseWorkspaceLine("/home/user/ws | ")
	if err == nil {
		t.Fatal("expected error for empty remote URL")
	}
}

func TestParseWorkspaceLine_InvalidFormat(t *testing.T) {
	_, err := parseWorkspaceLine("/home/user/ws")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestParseWorkspaceLine_EmptyPath(t *testing.T) {
	_, err := parseWorkspaceLine(" | https://github.com/user/ws.git")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

// ─── filterIgnoredProjects ───────────────────────────────────────────────────

func TestFilterIgnoredProjects_NoPatterns(t *testing.T) {
	projects := []*Project{
		{GitRepository: GitRepository{Path: "/root/a"}},
		{GitRepository: GitRepository{Path: "/root/b"}},
	}
	result := filterIgnoredProjects(projects, nil)
	if len(result) != 2 {
		t.Errorf("len = %d, want 2", len(result))
	}
}

func TestFilterIgnoredProjects_MatchesOne(t *testing.T) {
	projects := []*Project{
		{GitRepository: GitRepository{Path: "/root/ignore-me"}},
		{GitRepository: GitRepository{Path: "/root/keep-me"}},
	}
	result := filterIgnoredProjects(projects, []string{"ignore-me"})
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	if result[0].Path != "/root/keep-me" {
		t.Errorf("Path = %q, want %q", result[0].Path, "/root/keep-me")
	}
}

func TestFilterIgnoredProjects_MatchesAll(t *testing.T) {
	projects := []*Project{
		{GitRepository: GitRepository{Path: "/root/a"}},
		{GitRepository: GitRepository{Path: "/root/b"}},
	}
	result := filterIgnoredProjects(projects, []string{"/root/"})
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestFilterIgnoredProjects_InvalidRegexp(t *testing.T) {
	projects := []*Project{
		{GitRepository: GitRepository{Path: "/root/a"}},
	}
	// pattern invalide → ignoré silencieusement, aucun projet filtré
	result := filterIgnoredProjects(projects, []string{"[invalid"})
	if len(result) != 1 {
		t.Errorf("len = %d, want 1 (invalid regex should be skipped)", len(result))
	}
}

// ─── parseIgnoreFile ─────────────────────────────────────────────────────────

func TestParseIgnoreFile_FileNotExist(t *testing.T) {
	root := t.TempDir()
	patterns, err := parseIgnoreFile(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patterns) != 0 {
		t.Errorf("len = %d, want 0", len(patterns))
	}
}

func TestParseIgnoreFile_WithComments(t *testing.T) {
	root := t.TempDir()
	content := "# comment\npattern1\n\npattern2\n# another comment\npattern3\n"
	writeFile(t, filepath.Join(root, IgnoreFileName), content)

	patterns, err := parseIgnoreFile(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patterns) != 3 {
		t.Fatalf("len = %d, want 3", len(patterns))
	}
	if patterns[0] != "pattern1" || patterns[1] != "pattern2" || patterns[2] != "pattern3" {
		t.Errorf("patterns = %v", patterns)
	}
}

// ─── parseProjectsFile ───────────────────────────────────────────────────────

func TestParseProjectsFile_WithCommentsAndBlanks(t *testing.T) {
	root := t.TempDir()
	content := "# header\n\nprojectA | https://github.com/u/a.git\nprojectB | https://github.com/u/b.git # inline comment\n"
	writeFile(t, filepath.Join(root, ProjectsFileName), content)

	projects, err := parseProjectsFile(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("len = %d, want 2", len(projects))
	}
}

func TestParseProjectsFile_WithIgnoreFile(t *testing.T) {
	root := t.TempDir()
	projectsContent := "keep | https://github.com/u/keep.git\nignored | https://github.com/u/ignored.git\n"
	writeFile(t, filepath.Join(root, ProjectsFileName), projectsContent)
	writeFile(t, filepath.Join(root, IgnoreFileName), "ignored\n")

	projects, err := parseProjectsFile(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("len = %d, want 1", len(projects))
	}
	if projects[0].Name != "keep" {
		t.Errorf("Name = %q, want %q", projects[0].Name, "keep")
	}
}

// ─── parseWorkspacesFile ─────────────────────────────────────────────────────

func TestParseWorkspacesFile_FileNotExist(t *testing.T) {
	root := t.TempDir()
	workspaces, err := parseWorkspacesFile(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(workspaces) != 0 {
		t.Errorf("len = %d, want 0", len(workspaces))
	}
}

func TestParseWorkspacesFile_Valid(t *testing.T) {
	root := t.TempDir()
	content := "# comment\n/ws/alpha | https://github.com/u/alpha.git\n/ws/beta | https://github.com/u/beta.git\n"
	writeFile(t, filepath.Join(root, WorkspacesFileName), content)

	workspaces, err := parseWorkspacesFile(root)
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
