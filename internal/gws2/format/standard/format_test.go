package standard

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gws2"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

var fullDocuments = map[string]string{
	"yaml": `version: 1
self:
  remotes:
    - { name: origin, url: git@example.com:org/root.git }
workspaces:
  - path: sub
  - path: libs
    remotes:
      - { url: git@example.com:org/libs.git }
projects:
  - path: api
    remotes:
      - { name: origin, url: git@example.com:org/api.git }
      - { url: git@example.com:up/api.git }
settings:
  custom: value
`,
	"json": `{
  "version": 1,
  "self": {"remotes": [{"name": "origin", "url": "git@example.com:org/root.git"}]},
  "workspaces": [
    {"path": "sub"},
    {"path": "libs", "remotes": [{"url": "git@example.com:org/libs.git"}]}
  ],
  "projects": [
    {"path": "api", "remotes": [{"name": "origin", "url": "git@example.com:org/api.git"}, {"url": "git@example.com:up/api.git"}]}
  ],
  "settings": {"custom": "value"}
}
`,
	"toml": `version = 1

[self]
remotes = [{ name = "origin", url = "git@example.com:org/root.git" }]

[[workspaces]]
path = "sub"

[[workspaces]]
path = "libs"
remotes = [{ url = "git@example.com:org/libs.git" }]

[[projects]]
path = "api"
remotes = [{ name = "origin", url = "git@example.com:org/api.git" }, { url = "git@example.com:up/api.git" }]

[settings]
custom = "value"
`,
}

func assertFullDocument(t *testing.T, root string, node *gws2.Workspace) {
	t.Helper()
	if node.SelfEntry == nil || node.SelfEntry.GetOriginRemote().URL != "git@example.com:org/root.git" {
		t.Fatalf("self entry = %+v", node.SelfEntry)
	}
	if len(node.Children) != 2 {
		t.Fatalf("children = %d, want 2", len(node.Children))
	}
	if node.Children[0].IsGitRepository() {
		t.Fatal("workspace without remotes must be a folder")
	}
	if node.Children[1].GetOriginRemote().Name != "origin" {
		t.Fatalf("libs remote name = %q, want origin", node.Children[1].GetOriginRemote().Name)
	}
	if len(node.Projects) != 1 || node.Projects[0].GetPath() != filepath.Join(root, "api") {
		t.Fatalf("projects = %+v", node.Projects)
	}
	if up := node.Projects[0].GetUpstreamRemote(); up == nil || up.Name != "upstream" {
		t.Fatalf("api upstream = %+v", up)
	}
}

func TestRead_FullDocumentInEveryEncoding(t *testing.T) {
	for ext, content := range fullDocuments {
		t.Run(ext, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, gws2.ConfigDirName, FileBaseName+"."+ext), content)

			node, err := (Format{}).New(root).Read()
			if err != nil {
				t.Fatal(err)
			}
			assertFullDocument(t, root, node)
		})
	}
}

func TestWriteProjects_KeepsEncodingAndSettings(t *testing.T) {
	for ext, content := range fullDocuments {
		t.Run(ext, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, gws2.ConfigDirName, FileBaseName+"."+ext)
			writeFile(t, path, content)

			node, err := (Format{}).New(root).Read()
			if err != nil {
				t.Fatal(err)
			}
			node.AddProject(gws2.NewProject(root, "web", []*git.Remote{{Name: "origin", URL: "git@example.com:org/web.git"}}))
			if err := (Format{}).New(root).WriteProjects(node); err != nil {
				t.Fatal(err)
			}

			entries, err := os.ReadDir(filepath.Join(root, gws2.ConfigDirName))
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries[0].Name() != FileBaseName+"."+ext {
				t.Fatalf("config dir = %v, want only %s", entries, FileBaseName+"."+ext)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), "custom") || !strings.Contains(string(data), "web") {
				t.Fatalf("rewritten %s:\n%s", ext, data)
			}

			reread, err := (Format{}).New(root).Read()
			if err != nil {
				t.Fatal(err)
			}
			if len(reread.Projects) != 2 || len(reread.Children) != 2 || reread.SelfEntry == nil {
				t.Fatalf("after write: %d projects, %d children, self %v", len(reread.Projects), len(reread.Children), reread.SelfEntry)
			}
		})
	}
}

func TestRead_Errors(t *testing.T) {
	cases := map[string]map[string]string{
		"unknown version": {
			"yaml": "version: 2\n",
			"json": `{"version": 2}`,
			"toml": "version = 2\n",
		},
		"project without remote": {
			"yaml": "projects:\n  - path: api\n",
			"json": `{"projects": [{"path": "api"}]}`,
			"toml": "[[projects]]\npath = \"api\"\n",
		},
		"project without path": {
			"yaml": "projects:\n  - remotes: [{url: x}]\n",
			"json": `{"projects": [{"remotes": [{"url": "x"}]}]}`,
			"toml": "[[projects]]\nremotes = [{ url = \"x\" }]\n",
		},
		"remote without url": {
			"yaml": "workspaces:\n  - path: sub\n    remotes: [{name: origin}]\n",
			"json": `{"workspaces": [{"path": "sub", "remotes": [{"name": "origin"}]}]}`,
			"toml": "[[workspaces]]\npath = \"sub\"\nremotes = [{ name = \"origin\" }]\n",
		},
		"unknown key": {
			"yaml": "projets:\n  - path: api\n",
			"json": `{"projets": [{"path": "api"}]}`,
			"toml": "[[projets]]\npath = \"api\"\n",
		},
	}
	for name, byExt := range cases {
		for ext, content := range byExt {
			t.Run(name+"/"+ext, func(t *testing.T) {
				root := t.TempDir()
				writeFile(t, filepath.Join(root, FileBaseName+"."+ext), content)
				if _, err := (Format{}).New(root).Read(); err == nil {
					t.Fatal("expected error")
				}
			})
		}
	}
}

func TestLocate_Priority(t *testing.T) {
	if canRead(t.TempDir()) {
		t.Fatal("CanRead on empty dir = true")
	}

	root := t.TempDir()
	writeFile(t, filepath.Join(root, FileBaseName+".yaml"), "projects:\n  - path: rootyaml\n    remotes: [{url: x}]\n")
	writeFile(t, filepath.Join(root, gws2.ConfigDirName, FileBaseName+".toml"), "[[projects]]\npath = \"dirtoml\"\nremotes = [{ url = \"x\" }]\n")
	writeFile(t, filepath.Join(root, gws2.ConfigDirName, FileBaseName+".json"), `{"projects": [{"path": "dirjson", "remotes": [{"url": "x"}]}]}`)

	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	config, ok := (Format{}).Detect(root)
	if !ok {
		t.Fatal("Detect = false")
	}
	if strings.Count(logs.String(), "level=WARN") != 1 {
		t.Fatalf("Detect must warn exactly once, got:\n%s", logs.String())
	}

	node, err := config.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(node.Projects) != 1 || node.Projects[0].GetName() != "dirjson" {
		t.Fatalf("projects = %+v, want dirjson (.gws dir first, json before toml)", node.Projects)
	}
	warning := logs.String()
	if !strings.Contains(warning, FileBaseName+".toml") || !strings.Contains(warning, FileBaseName+".yaml") {
		t.Fatalf("expected a warning listing the ignored files, got:\n%s", warning)
	}
}

func TestWriteWorkspaces_NewWorkspaceIsYaml(t *testing.T) {
	root := t.TempDir()
	ws := gws2.NewRootWorkspace(root)
	ws.AddWorkspace(gws2.NewFolderWorkspace(root, "tools"))

	if err := (Format{}).New(root).WriteWorkspaces(ws); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(root, gws2.ConfigDirName, FileBaseName+".yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "version: 1") || !strings.Contains(string(data), "path: tools") {
		t.Fatalf("written file:\n%s", data)
	}
}

func TestClear_RemovesFileWhenEmpty(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, gws2.ConfigDirName, FileBaseName+".json")
	writeFile(t, path, `{"projects": [{"path": "api", "remotes": [{"url": "x"}]}], "workspaces": [{"path": "sub"}]}`)

	if removed, err := (Format{}).New(root).ClearProjects(); err != nil || removed != path {
		t.Fatalf("ClearProjects = %q, %v", removed, err)
	}
	node, err := (Format{}).New(root).Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(node.Projects) != 0 || len(node.Children) != 1 {
		t.Fatalf("after ClearProjects: %d projects, %d children", len(node.Projects), len(node.Children))
	}

	if _, err := (Format{}).New(root).ClearWorkspaces(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("empty file should be removed")
	}
}

func TestConfig_ReusesHandleAcrossWriteClearWrite(t *testing.T) {
	root := t.TempDir()
	config := (Format{}).New(root)
	ws := gws2.NewRootWorkspace(root)
	ws.AddProject(gws2.NewProject(root, "api", []*git.Remote{{Name: "origin", URL: "git@example.com:org/api.git"}}))

	if err := config.WriteProjects(ws); err != nil {
		t.Fatal(err)
	}
	node, err := config.Read()
	if err != nil || len(node.Projects) != 1 {
		t.Fatalf("read after write: %v, %+v", err, node)
	}
	if removed, err := config.ClearProjects(); err != nil || removed != config.Path() {
		t.Fatalf("ClearProjects = %q, %v", removed, err)
	}
	if _, err := os.Stat(config.Path()); !os.IsNotExist(err) {
		t.Fatal("file should be removed")
	}
	if err := config.WriteProjects(ws); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(config.Path()); err != nil {
		t.Fatalf("file should be recreated at %s: %v", config.Path(), err)
	}
}

func canRead(dir string) bool {
	_, ok := (Format{}).Detect(dir)
	return ok
}
