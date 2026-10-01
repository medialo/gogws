package gws2

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestHookConflict_PassesWithoutConflict(t *testing.T) {
	root := t.TempDir()
	hooks := filepath.Join(root, ConfigDirName, HooksDirName)
	writeFile(t, filepath.Join(hooks, "api.post-ff.sh"), "x")
	writeFile(t, filepath.Join(hooks, "windows", "api.post-ff.ps1"), "x")
	writeFile(t, filepath.Join(hooks, "post-ff.sh"), "x")

	w := &Workspace{Entry: Entry{AbsolutePath: root}}
	if got := DoctorRules[HookConflict].test(w); got != Passed {
		t.Fatalf("got %v, want Passed", got)
	}
}

func TestHookConflict_FailsAndExplains(t *testing.T) {
	root := t.TempDir()
	hooks := filepath.Join(root, ConfigDirName, HooksDirName)
	writeFile(t, filepath.Join(hooks, "api.post-ff.sh"), "x")
	writeFile(t, filepath.Join(hooks, "api.post-ff.js"), "x")
	writeFile(t, filepath.Join(hooks, "linux", "pre-ff"), "x")
	writeFile(t, filepath.Join(hooks, "linux", "pre-ff.py"), "x")

	w := &Workspace{Entry: Entry{AbsolutePath: root}}
	if got := DoctorRules[HookConflict].test(w); got != Failed {
		t.Fatalf("got %v, want Failed", got)
	}

	lines := DoctorRules[HookConflict].Explain(w)
	if len(lines) != 2 {
		t.Fatalf("got %v, want 2 lines", lines)
	}
	if !strings.HasPrefix(lines[0], "api.post-ff: ") || !strings.Contains(lines[0], "api.post-ff.js") || !strings.Contains(lines[0], "api.post-ff.sh") {
		t.Errorf("line 0 = %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "linux/pre-ff: ") || !strings.Contains(lines[1], "pre-ff.py") {
		t.Errorf("line 1 = %q", lines[1])
	}
}

func TestHookConflict_DoesNotInvalidateWorkspaceOnLoad(t *testing.T) {
	root := t.TempDir()
	hooks := filepath.Join(root, ConfigDirName, HooksDirName)
	writeFile(t, filepath.Join(hooks, "api.post-ff.sh"), "x")
	writeFile(t, filepath.Join(hooks, "api.post-ff.js"), "x")

	w := &Workspace{Entry: Entry{AbsolutePath: root}}
	if !w.IsValid() {
		t.Fatal("a hook conflict must not make the workspace invalid on load")
	}
}

func TestHookConflict_NoHooksDir(t *testing.T) {
	w := &Workspace{Entry: Entry{AbsolutePath: t.TempDir()}}
	if got := DoctorRules[HookConflict].test(w); got != Passed {
		t.Fatalf("got %v, want Passed", got)
	}
}
