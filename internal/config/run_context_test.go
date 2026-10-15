package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/gws2/loader"
)

func TestResolveWorkspaceRoot_WorkingDirOutsideWorkspace(t *testing.T) {
	outside := t.TempDir()
	target := t.TempDir()

	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := os.Chdir(outside); err != nil {
		t.Fatal(err)
	}

	got, global, err := resolveWorkspaceRoot(target, loader.ModeNearest, false)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(target)
	if got != want || global {
		t.Fatalf("root = %q global = %v, want %q", got, global, want)
	}
}

func setHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func globalWorkspacesFile(t *testing.T, home string) string {
	t.Helper()
	return filepath.Join(home, UserConfigDir, GlobalWorkspacesFile)
}

func writeGlobalWorkspaces(t *testing.T, home string) {
	t.Helper()
	dir := filepath.Join(home, UserConfigDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, GlobalWorkspacesFile), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveWorkspaceRoot_TopmostFromWorkingDir(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "inner")
	for _, dir := range []string{root, inner} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".projects.gws"), []byte("api | git@example.com:org/api.git\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, global, err := resolveWorkspaceRoot(inner, loader.ModeTopmost, false)
	if err != nil {
		t.Fatal(err)
	}
	if gws2.PathKey(got) != gws2.PathKey(root) || global {
		t.Fatalf("root = %q global = %v, want %q", got, global, root)
	}
}

func TestResolveWorkspaceRoot_GlobalMode(t *testing.T) {
	home := setHome(t)

	want := filepath.Dir(globalWorkspacesFile(t, home))

	got, global, err := resolveWorkspaceRoot(t.TempDir(), loader.ModeGlobal, false)
	if err != nil {
		t.Fatal(err)
	}
	if gws2.PathKey(got) != gws2.PathKey(want) || !global {
		t.Fatalf("root = %q global = %v, want %q global", got, global, want)
	}
}

func TestResolveWorkspaceRoot_GlobalFallback(t *testing.T) {
	home := setHome(t)
	outside := t.TempDir()

	if _, _, err := resolveWorkspaceRoot(outside, loader.ModeTopmost, true); !errors.Is(err, loader.ErrNoWorkspace) {
		t.Fatalf("without global file: err = %v, want ErrNoWorkspace", err)
	}

	writeGlobalWorkspaces(t, home)

	if _, _, err := resolveWorkspaceRoot(outside, loader.ModeTopmost, false); !errors.Is(err, loader.ErrNoWorkspace) {
		t.Fatalf("fallback disabled: err = %v, want ErrNoWorkspace", err)
	}

	got, global, err := resolveWorkspaceRoot(outside, loader.ModeTopmost, true)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Dir(globalWorkspacesFile(t, home))
	if gws2.PathKey(got) != gws2.PathKey(want) || !global {
		t.Fatalf("root = %q global = %v, want %q global", got, global, want)
	}
}
