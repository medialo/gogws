package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initRepo(t *testing.T, path string, remoteURL string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	commands := [][]string{{"init", "-q"}}
	if remoteURL != "" {
		commands = append(commands, []string{"remote", "add", "origin", remoteURL})
	}
	for _, args := range commands {
		cmd := exec.Command("git", args...)
		cmd.Dir = path
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
}

func TestDiscoverRepositoryOriginFirst(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	root := t.TempDir()
	path := filepath.Join(root, "multi")
	initRepo(t, path, "")
	for _, args := range [][]string{
		{"remote", "add", "upstream", "https://example.com/up.git"},
		{"remote", "add", "origin", "https://example.com/origin.git"},
		{"remote", "add", "backup", "https://example.com/backup.git"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = path
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}

	repo, err := DiscoverRepository(root, path)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, r := range repo.Remotes {
		names = append(names, r.Name)
	}
	want := []string{"origin", "backup", "upstream"}
	if len(names) != len(want) {
		t.Fatalf("remotes = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("remotes = %v, want %v", names, want)
		}
	}
}

func TestDiscoverRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	root := t.TempDir()
	initRepo(t, filepath.Join(root, "libs", "a"), "https://example.com/a.git")
	initRepo(t, filepath.Join(root, "noremote"), "")
	if err := os.MkdirAll(filepath.Join(root, "b"), 0755); err != nil {
		t.Fatal(err)
	}

	repo, err := DiscoverRepository(root, filepath.Join(root, "libs", "a"))
	if err != nil {
		t.Fatal(err)
	}
	if repo == nil {
		t.Fatal("expected repository libs/a to be found")
	}
	if repo.Path != filepath.Join("libs", "a") {
		t.Errorf("Path = %q, want %q", repo.Path, filepath.Join("libs", "a"))
	}
	if len(repo.Remotes) != 1 || repo.Remotes[0].Name != "origin" || repo.Remotes[0].URL != "https://example.com/a.git" {
		t.Errorf("Remotes = %+v", repo.Remotes)
	}

	repo, err = DiscoverRepository(root, filepath.Join(root, "noremote"))
	if err != nil {
		t.Fatal(err)
	}
	if repo == nil || len(repo.Remotes) != 0 {
		t.Errorf("noremote = %+v, want found with no remotes", repo)
	}

	for _, path := range []string{filepath.Join(root, "b"), root} {
		repo, err := DiscoverRepository(root, path)
		if err != nil {
			t.Fatal(err)
		}
		if repo != nil {
			t.Errorf("DiscoverRepository(%q) = %+v, want nil", path, repo)
		}
	}
}
