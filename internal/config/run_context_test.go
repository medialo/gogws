package config

import (
	"os"
	"path/filepath"
	"testing"
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

	got, err := resolveWorkspaceRoot(target)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(target)
	if got != want {
		t.Fatalf("root = %q, want %q", got, want)
	}
}
