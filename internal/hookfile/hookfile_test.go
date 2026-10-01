package hookfile

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestParse(t *testing.T) {
	cases := []struct {
		name   string
		ok     bool
		target string
		hook   string
		ext    string
	}{
		{"post-ff", true, "", "post-ff", ""},
		{"post-ff.sh", true, "", "post-ff", "sh"},
		{"api.post-ff.ps1", true, "api", "post-ff", "ps1"},
		{"foo.bar.post-ff.sh", true, "foo.bar", "post-ff", "sh"},
		{"windows.post-ff", true, "windows", "post-ff", ""},
		{"post-ff.sample", false, "", "", ""},
		{"post-ff.txt", false, "", "", ""},
		{"api.post-fff.sh", false, "", "", ""},
		{"README.md", false, "", "", ""},
	}
	for _, c := range cases {
		f, ok := Parse(filepath.Join("dir", c.name))
		if ok != c.ok {
			t.Errorf("%s: ok = %v, want %v", c.name, ok, c.ok)
			continue
		}
		if ok && (f.Target != c.target || f.Hook != c.hook || f.Ext != c.ext) {
			t.Errorf("%s: got %+v, want target=%q hook=%q ext=%q", c.name, f, c.target, c.hook, c.ext)
		}
	}
}

func TestSearchDirs(t *testing.T) {
	if got := SearchDirs("h", "windows"); len(got) != 2 || got[0] != filepath.Join("h", "windows") || got[1] != "h" {
		t.Errorf("windows: got %v", got)
	}
	if got := SearchDirs("h", "freebsd"); len(got) != 1 || got[0] != "h" {
		t.Errorf("freebsd: got %v", got)
	}
}

func TestResolve_OSDirWinsOverCatchAll(t *testing.T) {
	base := t.TempDir()
	touch(t, filepath.Join(base, "api.post-ff.sh"))
	touch(t, filepath.Join(base, "windows", "api.post-ff.ps1"))

	got, err := Resolve(Scan, SearchDirs(base, "windows"), "api", "post-ff")
	if err != nil || got == nil || filepath.Base(got.Path) != "api.post-ff.ps1" {
		t.Fatalf("windows: got %+v, %v; want windows/api.post-ff.ps1", got, err)
	}

	got, err = Resolve(Scan, SearchDirs(base, "linux"), "api", "post-ff")
	if err != nil || got == nil || filepath.Base(got.Path) != "api.post-ff.sh" {
		t.Fatalf("linux: got %+v, %v; want catch-all api.post-ff.sh", got, err)
	}
}

func TestResolve_ConflictInSameDir(t *testing.T) {
	base := t.TempDir()
	touch(t, filepath.Join(base, "api.post-ff"))
	touch(t, filepath.Join(base, "api.post-ff.sh"))

	_, err := Resolve(Scan, SearchDirs(base, "linux"), "api", "post-ff")
	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("err = %v; want ConflictError", err)
	}
	if len(conflict.Files) != 2 || !strings.Contains(err.Error(), "api.post-ff.sh") || !strings.Contains(err.Error(), "doctor") {
		t.Fatalf("unexpected conflict: %v", err)
	}
}

func TestResolve_OSConflictShadowsNothingElse(t *testing.T) {
	base := t.TempDir()
	touch(t, filepath.Join(base, "windows", "api.post-ff.ps1"))
	touch(t, filepath.Join(base, "windows", "api.post-ff.cmd"))
	touch(t, filepath.Join(base, "api.post-ff.sh"))

	if _, err := Resolve(Scan, SearchDirs(base, "windows"), "api", "post-ff"); err == nil {
		t.Fatal("windows: want a conflict")
	}
	if got, err := Resolve(Scan, SearchDirs(base, "darwin"), "api", "post-ff"); err != nil || got == nil {
		t.Fatalf("darwin: got %+v, %v; want catch-all", got, err)
	}
}

func TestResolve_NoMatch(t *testing.T) {
	base := t.TempDir()
	touch(t, filepath.Join(base, "web.post-ff.sh"))
	got, err := Resolve(Scan, SearchDirs(base, "linux"), "api", "post-ff")
	if err != nil || got != nil {
		t.Fatalf("got %+v, %v; want nothing", got, err)
	}
}

func TestConflicts_GroupedAndSorted(t *testing.T) {
	base := t.TempDir()
	for _, n := range []string{"web.post-ff.js", "web.post-ff.sh", "api.pre-ff", "api.pre-ff.py", "api.post-ff.sh", "post-fetch.ps1"} {
		touch(t, filepath.Join(base, n))
	}
	files, err := Scan(base)
	if err != nil {
		t.Fatal(err)
	}
	got := Conflicts(base, files)
	if len(got) != 2 || got[0].Key.String() != "api.pre-ff" || got[1].Key.String() != "web.post-ff" {
		t.Fatalf("got %+v", got)
	}
}

func TestScan_MissingDirIsEmpty(t *testing.T) {
	files, err := Scan(filepath.Join(t.TempDir(), "nope"))
	if err != nil || len(files) != 0 {
		t.Fatalf("got %v, %v", files, err)
	}
}
