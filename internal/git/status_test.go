package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseBranches(t *testing.T) {
	output := []byte("main|origin/main|*|ahead 2, behind 1\nfeature|origin/feature| |behind 3\nlocal|| |\nold|origin/old| |gone\nsynced|origin/synced| |\n")

	branches := parseBranches(output)
	if len(branches) != 5 {
		t.Fatalf("branches = %d, want 5", len(branches))
	}

	cases := []struct {
		name          string
		current       bool
		ahead, behind int
	}{
		{"main", true, 2, 1},
		{"feature", false, 0, 3},
		{"local", false, 0, 0},
		{"old", false, 0, 0},
		{"synced", false, 0, 0},
	}
	for i, c := range cases {
		b := branches[i]
		if b.Name != c.name || b.IsCurrent != c.current || b.Ahead != c.ahead || b.Behind != c.behind {
			t.Errorf("branch %d = %+v, want %+v", i, b, c)
		}
	}
	if branches[2].Upstream != "" {
		t.Errorf("local upstream = %q, want empty", branches[2].Upstream)
	}
}

func TestParsePorcelainStatus(t *testing.T) {
	output := []byte("# branch.oid abc123\n# branch.head main\n# branch.upstream origin/main\n# branch.ab +4 -2\n1 .M N... 100644 100644 100644 a b file.go\n? new.txt\n? other.txt\n")

	status := &RepositoryStatus{}
	parsePorcelainStatus(output, status)

	if status.Oid != "abc123" || status.Branch != "main" || !status.HasRemote {
		t.Fatalf("header = %+v", status)
	}
	if status.Ahead != 4 || status.Behind != 2 {
		t.Fatalf("ahead/behind = %d/%d, want 4/2", status.Ahead, status.Behind)
	}
	if status.Uncommitted != 1 || status.Untracked != 2 || status.Clean {
		t.Fatalf("changes = %d uncommitted, %d untracked, clean %v", status.Uncommitted, status.Untracked, status.Clean)
	}
}

func TestIsRepository(t *testing.T) {
	root := t.TempDir()
	if IsRepository(root) {
		t.Fatal("empty dir reported as repository")
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !IsRepository(root) {
		t.Fatal("dir with .git not reported as repository")
	}
}
