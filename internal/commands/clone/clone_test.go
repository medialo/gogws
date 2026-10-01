package clone

import (
	"path/filepath"
	"testing"

	"github.com/medialo/gogws/internal/gws2"
)

func TestFindProject(t *testing.T) {
	root := t.TempDir()
	api := &gws2.Project{GitRepository: gws2.GitRepository{Entry: gws2.Entry{AbsolutePath: filepath.Join(root, "api"), RelativePath: "api", Name: "api"}}}
	lib := &gws2.Project{GitRepository: gws2.GitRepository{Entry: gws2.Entry{AbsolutePath: filepath.Join(root, "libs", "shared"), RelativePath: "libs/shared", Name: "shared"}}}
	ws := &gws2.Workspace{Entry: gws2.Entry{AbsolutePath: root}, Projects: []*gws2.Project{api, lib}}

	cases := map[string]*gws2.Project{
		"api":                                 api,
		"libs/shared":                         lib,
		"libs/shared/":                        lib,
		filepath.Join(root, "libs", "shared"): lib,
		"shared":                              nil,
		"nope":                                nil,
	}
	for arg, want := range cases {
		if got := findProject(ws, arg); got != want {
			t.Errorf("findProject(%q) = %v, want %v", arg, got, want)
		}
	}
}
