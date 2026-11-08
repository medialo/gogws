package config

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTrustedHook_PathCaseOnWindows(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	path := filepath.Join(home, "Work", "WS", ".gws", "hooks", "post-ff.ps1")
	if err := TrustHook(path, "abc"); err != nil {
		t.Fatal(err)
	}

	other := strings.ToLower(path)
	_, found := FindTrustedHook(other)
	if want := runtime.GOOS == "windows"; found != want {
		t.Fatalf("FindTrustedHook(%q) found = %v, want %v", other, found, want)
	}

	if err := TrustHook(other, "def"); err != nil {
		t.Fatal(err)
	}
	hooks, err := LoadTrustedHooks()
	if err != nil {
		t.Fatal(err)
	}
	wantCount := 2
	if runtime.GOOS == "windows" {
		wantCount = 1
	}
	if len(hooks) != wantCount {
		t.Fatalf("got %d trusted hooks, want %d: %+v", len(hooks), wantCount, hooks)
	}
}
