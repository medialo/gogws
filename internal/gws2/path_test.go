package gws2

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsAnchoredPath(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "abs")
	cases := map[string]bool{
		abs:         true,
		"~":         true,
		"~/dev/api": true,
		"./name":    false,
		"name":      false,
		"sub/api":   false,
	}
	for path, want := range cases {
		if got := IsAnchoredPath(path); got != want {
			t.Fatalf("IsAnchoredPath(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestResolvePath(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(t.TempDir(), "abs")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	cases := []struct {
		name string
		path string
		want string
	}{
		{"relative", "sub/api", filepath.Join(root, "sub", "api")},
		{"dot", ".", root},
		{"absolute", abs, abs},
		{"home", "~", home},
		{"under home", "~/dev/api", filepath.Join(home, "dev", "api")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ResolvePath(root, c.path); PathKey(got) != PathKey(c.want) {
				t.Fatalf("ResolvePath(%q) = %q, want %q", c.path, got, c.want)
			}
		})
	}
}
