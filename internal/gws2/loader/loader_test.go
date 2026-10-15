package loader

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/gws2/format/gws"
	"github.com/medialo/gogws/internal/gws2/format/standard"
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

func captureWarnings(t *testing.T) *strings.Builder {
	t.Helper()
	var logs strings.Builder
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &logs
}

func assertSamePath(t *testing.T, got, want string) {
	t.Helper()
	gotEval, _ := filepath.EvalSymlinks(got)
	wantEval, _ := filepath.EvalSymlinks(want)
	if !strings.EqualFold(gotEval, wantEval) {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestLoad_SelfLineIsNotAChildWorkspace(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, gws.WorkspacesFileName), ". | git@example.com:org/root.git\nsub | folder\n")
	writeFile(t, filepath.Join(root, "sub", gws.WorkspacesFileName), ". | folder\n")

	ws, err := NewFromPath(root).Load()
	if err != nil {
		t.Fatal(err)
	}

	all := append([]*gws2.Workspace{ws}, ws.FlattenWorkspaces()...)
	if len(all) != 2 {
		paths := make([]string, len(all))
		for i, w := range all {
			paths[i] = w.GetPath()
		}
		t.Fatalf("got %d workspaces %v, want root + sub", len(all), paths)
	}
	if ws.SelfEntry == nil || ws.SelfEntry.GetOriginRemote().URL != "git@example.com:org/root.git" {
		t.Fatalf("root self entry not kept: %+v", ws.SelfEntry)
	}
	if got, ok := ws.Index().Get(root); !ok || got != gws2.Repository(ws) {
		t.Fatalf("index entry for root = %v, want the root workspace", got)
	}
	if !ws.IsGitRepository() || ws.GetOriginRemote().URL != "git@example.com:org/root.git" {
		t.Fatalf("root must take its remote from its own '.' line, got %+v", ws.Remotes)
	}
	sub := ws.Children[0]
	if sub.IsGitRepository() {
		t.Fatalf("sub declared as folder by its parent must stay a folder, got %+v", sub.Remotes)
	}
}

func TestLoad_MissingProjectsFileIsNotAWarning(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, gws2.ConfigDirName, gws.WorkspacesFileNameInDir), "sub | folder\n")
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	logs := captureWarnings(t)

	ws, err := NewFromPath(root).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(ws.Projects) != 0 || len(ws.Children) != 1 {
		t.Fatalf("got %d projects, %d children", len(ws.Projects), len(ws.Children))
	}
	if logs.Len() > 0 {
		t.Fatalf("unexpected warnings:\n%s", logs.String())
	}
}

func TestLoad_LegacyFilesWithoutConfigDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, gws.ProjectsFileName), "# legacy\napi | git@example.com:org/api.git\nlibs/shared | git@example.com:org/shared.git | git@example.com:up/shared.git upstream\n")
	writeFile(t, filepath.Join(root, gws.WorkspacesFileName), ". | git@example.com:org/root.git\ntools | folder\n")
	writeFile(t, filepath.Join(root, "tools", gws.ProjectsFileName), "cli | git@example.com:org/cli.git\n")

	if _, err := os.Stat(filepath.Join(root, gws2.ConfigDirName)); !os.IsNotExist(err) {
		t.Fatalf("test setup must not create %s", gws2.ConfigDirName)
	}

	ws, err := NewFromPath(root).Load()
	if err != nil {
		t.Fatal(err)
	}

	if len(ws.Projects) != 2 {
		t.Fatalf("root projects = %d, want 2", len(ws.Projects))
	}
	shared := ws.Projects[1]
	if shared.GetPath() != filepath.Join(root, "libs", "shared") || len(shared.Remotes) != 2 || shared.Remotes[1].Name != "upstream" {
		t.Fatalf("libs/shared = %+v, remotes %+v", shared.Entry, shared.Remotes)
	}
	if !ws.IsGitRepository() {
		t.Fatal("root '.' line from the legacy workspaces file must be read")
	}
	if len(ws.Children) != 1 || ws.Children[0].GetName() != "tools" {
		t.Fatalf("children = %+v", ws.Children)
	}
	tools := ws.Children[0]
	if len(tools.Projects) != 1 || tools.Projects[0].GetPath() != filepath.Join(root, "tools", "cli") {
		t.Fatalf("tools projects = %+v", tools.Projects)
	}
}

func TestLoad_ConfigDirWinsOverLegacyFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, gws.ProjectsFileName), "legacy | git@example.com:org/legacy.git\n")
	writeFile(t, filepath.Join(root, gws2.ConfigDirName, gws.ProjectsFileNameInDir), "api | git@example.com:org/api.git\n")

	ws, err := NewFromPath(root).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(ws.Projects) != 1 || ws.Projects[0].GetName() != "api" {
		t.Fatalf("projects = %+v, want only api from %s", ws.Projects, gws2.ConfigDirName)
	}
}

func TestLoad_AppliesIgnoreFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, gws2.ConfigDirName, standard.FileBaseName+".yaml"), "version: 1\nprojects:\n  - path: keep\n    remotes: [{url: git@example.com:org/keep.git}]\n  - path: ignored\n    remotes: [{url: git@example.com:org/ignored.git}]\n")
	writeFile(t, filepath.Join(root, gws.IgnoreFileName), "ignored\n")

	ws, err := NewFromPath(root).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(ws.Projects) != 1 || ws.Projects[0].GetName() != "keep" {
		t.Fatalf("projects = %+v, want only keep", ws.Projects)
	}
}

func TestLoad_YamlRootWithLegacyChild(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, gws2.ConfigDirName, standard.FileBaseName+".yaml"), "version: 1\nself:\n  remotes: [{url: git@example.com:org/root.git}]\nworkspaces:\n  - path: legacy\nprojects:\n  - path: api\n    remotes: [{url: git@example.com:org/api.git}]\n")
	writeFile(t, filepath.Join(root, "legacy", gws.ProjectsFileName), "cli | git@example.com:org/cli.git\n")

	ws, err := NewFromPath(root).Load()
	if err != nil {
		t.Fatal(err)
	}
	if ws.Config().Format().Name() != "standard" || !ws.IsGitRepository() {
		t.Fatalf("root format = %s, git = %v", ws.Config().Format().Name(), ws.IsGitRepository())
	}
	if len(ws.Projects) != 1 || len(ws.Children) != 1 {
		t.Fatalf("root: %d projects, %d children", len(ws.Projects), len(ws.Children))
	}
	legacy := ws.Children[0]
	if legacy.Config().Format().Name() != "gws" || len(legacy.Projects) != 1 {
		t.Fatalf("legacy child format = %s, projects = %d", legacy.Config().Format().Name(), len(legacy.Projects))
	}
}

func TestLoad_YamlWinsOverGwsInSameDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, standard.FileBaseName+".yaml"), "version: 1\nprojects:\n  - path: fromyaml\n    remotes: [{url: git@example.com:org/y.git}]\n")
	writeFile(t, filepath.Join(root, gws.ProjectsFileName), "fromgws | git@example.com:org/g.git\n")

	logs := captureWarnings(t)

	ws, err := NewFromPath(root).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(ws.Projects) != 1 || ws.Projects[0].GetName() != "fromyaml" {
		t.Fatalf("projects = %+v, want only fromyaml", ws.Projects)
	}
	if !strings.Contains(logs.String(), gws.ProjectsFileName) {
		t.Fatalf("expected a warning about the ignored gws configuration, got:\n%s", logs.String())
	}
}

func TestSave_KeepsOriginalFormat(t *testing.T) {
	root := t.TempDir()
	legacyFile := filepath.Join(root, gws.ProjectsFileName)
	writeFile(t, legacyFile, "api | git@example.com:org/api.git\n")

	ws, err := NewFromPath(root).Recursive(false).Load()
	if err != nil {
		t.Fatal(err)
	}
	ws.AddProject(gws2.NewProject(root, "web", []*git.Remote{{Name: "origin", URL: "git@example.com:org/web.git"}}))
	if err := ws.SaveProjects(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(legacyFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "web | git@example.com:org/web.git") {
		t.Fatalf("legacy file = %q", string(data))
	}
	if _, err := os.Stat(filepath.Join(root, gws2.ConfigDirName, standard.FileBaseName+".yaml")); !os.IsNotExist(err) {
		t.Fatal("saving a legacy workspace must not create a yaml file")
	}
}

func TestSave_NewWorkspaceIsBornInYaml(t *testing.T) {
	root := t.TempDir()

	ws, err := NewFromPath(root).RunDoctor(false).Recursive(false).Load()
	if err != nil {
		t.Fatal(err)
	}
	ws.AddProject(gws2.NewProject(root, "api", []*git.Remote{{Name: "origin", URL: "git@example.com:org/api.git"}}))
	if err := ws.SaveProjects(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(root, gws2.ConfigDirName, standard.FileBaseName+".yaml")); err != nil {
		t.Fatalf("expected %s to be created: %v", standard.FileBaseName+".yaml", err)
	}
	reloaded, err := NewFromPath(root).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Projects) != 1 || reloaded.Projects[0].GetName() != "api" {
		t.Fatalf("reloaded projects = %+v", reloaded.Projects)
	}
}

func TestSave_MissingChildGetsAConfig(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, gws2.ConfigDirName, standard.FileBaseName+".yaml"), "version: 1\nworkspaces:\n  - path: missing\n")

	ws, err := NewFromPath(root).Load()
	if err != nil {
		t.Fatal(err)
	}
	child := ws.Children[0]
	if child.FolderExist() || child.Config() == nil {
		t.Fatalf("missing child: exists=%v format=%v", child.FolderExist(), child.Config())
	}
}

func TestFindRoot_LegacyProjectsFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, gws.ProjectsFileName), "api | git@example.com:org/api.git\n")
	sub := filepath.Join(root, "api", "src")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindRoot(sub, ModeNearest)
	if err != nil {
		t.Fatal(err)
	}
	assertSamePath(t, got, root)
}

func TestFindRoot_WorkspacesFileInConfigDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, gws2.ConfigDirName, gws.WorkspacesFileNameInDir), "sub | folder\n")
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindRoot(sub, ModeNearest)
	if err != nil {
		t.Fatal(err)
	}
	assertSamePath(t, got, root)
}

func TestFindRoot_YamlAtRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, standard.FileBaseName+".yaml"), "version: 1\nworkspaces:\n  - path: sub\n")
	sub := filepath.Join(root, "sub", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindRoot(sub, ModeNearest)
	if err != nil {
		t.Fatal(err)
	}
	assertSamePath(t, got, root)
}

func TestFindRoot_NearestStopsAtProjectsOnlyOrWorkspacesOnly(t *testing.T) {
	for _, file := range []string{gws.ProjectsFileName, gws.WorkspacesFileName} {
		t.Run(file, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, gws.WorkspacesFileName), "inner | folder\n")
			inner := filepath.Join(root, "inner")
			content := "api | git@example.com:org/api.git\n"
			if file == gws.WorkspacesFileName {
				content = "sub | folder\n"
			}
			writeFile(t, filepath.Join(inner, file), content)
			start := filepath.Join(inner, "deep")
			if err := os.MkdirAll(start, 0o755); err != nil {
				t.Fatal(err)
			}

			got, err := FindRoot(start, ModeNearest)
			if err != nil {
				t.Fatal(err)
			}
			assertSamePath(t, got, inner)
		})
	}
}

func TestFindRoot_Topmost(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, gws2.ConfigDirName, standard.FileBaseName+".yaml"), "version: 1\nworkspaces:\n  - path: middle\n")
	middle := filepath.Join(root, "middle")
	writeFile(t, filepath.Join(middle, gws.WorkspacesFileName), "inner | folder\n")
	inner := filepath.Join(middle, "inner")
	writeFile(t, filepath.Join(inner, gws.ProjectsFileName), "api | git@example.com:org/api.git\n")

	got, err := FindRoot(inner, ModeTopmost)
	if err != nil {
		t.Fatal(err)
	}
	assertSamePath(t, got, root)

	got, err = FindRoot(inner, ModeNearest)
	if err != nil {
		t.Fatal(err)
	}
	assertSamePath(t, got, inner)
}

func TestFindRoot_NoWorkspace(t *testing.T) {
	dir := t.TempDir()
	for _, mode := range []WorkspaceMode{ModeNearest, ModeTopmost} {
		if _, err := FindRoot(dir, mode); !errors.Is(err, ErrNoWorkspace) {
			t.Fatalf("%s: err = %v, want ErrNoWorkspace", mode, err)
		}
	}
}

func TestNewGlobal_RejectsRelativePaths(t *testing.T) {
	home := t.TempDir()
	file := filepath.Join(home, gws2.ConfigDirName, "global.workspaces.yaml")
	for name, content := range map[string]string{
		"project":   "version: 1\nprojects:\n  - path: dev/api\n    url: git@example.com:org/api.git\n",
		"workspace": "version: 1\nworkspaces:\n  - path: dev\n",
		"folder":    "version: 1\nworkspaces:\n  - folder: ./dev\n",
	} {
		t.Run(name, func(t *testing.T) {
			writeFile(t, file, content)
			if _, err := standard.NewGlobal(filepath.Dir(file), file).Read(); err == nil {
				t.Fatal("expected an error for a relative path in the global file")
			}
		})
	}
}

func TestNewGlobal_LoadsHomeAndAbsolutePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	elsewhere := t.TempDir()
	file := filepath.Join(home, gws2.ConfigDirName, "global.workspaces.yaml")
	writeFile(t, file, "version: 1\nprojects:\n  - path: ~/dev/api\n    url: git@example.com:org/api.git\n  - path: "+filepath.ToSlash(filepath.Join(elsewhere, "web"))+"\n    remotes: [{url: git@example.com:org/web.git}]\nworkspaces:\n  - folder: "+filepath.ToSlash(elsewhere)+"\n")

	ws, err := NewGlobal(filepath.Dir(file), file).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(ws.Projects) != 2 {
		t.Fatalf("projects = %d, want 2", len(ws.Projects))
	}
	assertSamePathNoEval(t, ws.Projects[0].GetPath(), filepath.Join(home, "dev", "api"))
	assertSamePathNoEval(t, ws.Projects[1].GetPath(), filepath.Join(elsewhere, "web"))
	if ws.Config().Path() != file {
		t.Fatalf("config path = %s, want %s", ws.Config().Path(), file)
	}
	if len(ws.Children) != 1 {
		t.Fatalf("children = %d, want 1", len(ws.Children))
	}
	assertSamePathNoEval(t, ws.Children[0].AbsolutePath, elsewhere)
	if len(ws.Projects[0].Remotes) != 1 || ws.Projects[0].Remotes[0].Name != "origin" {
		t.Fatalf("url shorthand must yield a single origin remote, got %+v", ws.Projects[0].Remotes)
	}
}

func TestNewGlobal_MissingFileIsCreatedOnSave(t *testing.T) {
	home := t.TempDir()
	file := filepath.Join(home, gws2.ConfigDirName, "global.workspaces.yaml")

	ws, err := NewGlobal(home, file).RunDoctor(false).Recursive(false).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(ws.Projects) != 0 || len(ws.Children) != 0 {
		t.Fatalf("got %d projects, %d children", len(ws.Projects), len(ws.Children))
	}
	ws.AddProject(gws2.NewProject(home, filepath.Join(home, "api"), []*git.Remote{{Name: "origin", URL: "git@example.com:org/api.git"}}))
	if err := ws.SaveProjects(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("expected %s to be created: %v", file, err)
	}
	if _, err := os.Stat(filepath.Join(home, gws2.ConfigDirName, standard.FileBaseName+".yaml")); !os.IsNotExist(err) {
		t.Fatal("global save must not create a local workspaces file")
	}
}

func assertSamePathNoEval(t *testing.T, got, want string) {
	t.Helper()
	if gws2.PathKey(got) != gws2.PathKey(want) {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestLoad_JsonRootWithLegacyChild(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, gws2.ConfigDirName, standard.FileBaseName+".json"), `{"version": 1, "workspaces": [{"path": "legacy"}], "projects": [{"path": "api", "remotes": [{"url": "git@example.com:org/api.git"}]}]}`)
	writeFile(t, filepath.Join(root, "legacy", gws.ProjectsFileName), "cli | git@example.com:org/cli.git\n")

	ws, err := NewFromPath(root).Load()
	if err != nil {
		t.Fatal(err)
	}
	if ws.Config().Format().Name() != "standard" || len(ws.Projects) != 1 || len(ws.Children) != 1 {
		t.Fatalf("root format = %s, %d projects, %d children", ws.Config().Format().Name(), len(ws.Projects), len(ws.Children))
	}
	if legacy := ws.Children[0]; legacy.Config().Format().Name() != "gws" || len(legacy.Projects) != 1 {
		t.Fatalf("legacy child format = %s, projects = %d", legacy.Config().Format().Name(), len(legacy.Projects))
	}
}

func TestFindRoot_TomlAtRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, standard.FileBaseName+".toml"), "[[workspaces]]\npath = \"sub\"\n")
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindRoot(sub, ModeNearest)
	if err != nil {
		t.Fatal(err)
	}
	assertSamePath(t, got, root)
}

type countingFormat struct {
	gws2.WorkspaceFormat
	calls map[string]int
}

func (c *countingFormat) Detect(files gws2.ConfigFiles) (gws2.WorkspaceConfig, bool) {
	c.calls[gws2.PathKey(files.Dir)]++
	return c.WorkspaceFormat.Detect(files)
}

func withCountingFormats(t *testing.T) *countingFormat {
	t.Helper()
	counter := &countingFormat{WorkspaceFormat: standard.Format{}, calls: map[string]int{}}
	previousFormats := formats
	detectedMu.Lock()
	previousDetected := detected
	detected = map[string]detection{}
	detectedMu.Unlock()
	formats = []gws2.WorkspaceFormat{counter, gws.Format{}}
	t.Cleanup(func() {
		formats = previousFormats
		detectedMu.Lock()
		detected = previousDetected
		detectedMu.Unlock()
	})
	return counter
}

func TestDetect_OncePerDirectory(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, gws2.ConfigDirName, standard.FileBaseName+".yaml"), "version: 1\nworkspaces:\n  - path: sub\nprojects:\n  - path: api\n    remotes: [{url: git@example.com:org/api.git}]\n")
	writeFile(t, filepath.Join(root, "sub", standard.FileBaseName+".json"), `{"projects": [{"path": "cli", "remotes": [{"url": "git@example.com:org/cli.git"}]}]}`)
	counter := withCountingFormats(t)

	found, err := FindRoot(root, ModeNearest)
	if err != nil {
		t.Fatal(err)
	}
	ws, err := NewFromPath(found).Load()
	if err != nil {
		t.Fatal(err)
	}
	ws.AddProject(gws2.NewProject(found, "web", []*git.Remote{{Name: "origin", URL: "git@example.com:org/web.git"}}))
	if err := ws.SaveProjects(); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFromPath(found).Load(); err != nil {
		t.Fatal(err)
	}

	for _, dir := range []string{root, filepath.Join(root, "sub")} {
		if got := counter.calls[gws2.PathKey(dir)]; got != 1 {
			t.Fatalf("Detect(%s) called %d times, want 1", dir, got)
		}
	}
}

func TestDetect_ConflictWarnedOnce(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, standard.FileBaseName+".yaml"), "version: 1\nprojects:\n  - path: api\n    remotes: [{url: x}]\n")
	writeFile(t, filepath.Join(root, gws.ProjectsFileName), "legacy | y\n")
	withCountingFormats(t)

	logs := captureWarnings(t)

	if _, err := FindRoot(root, ModeNearest); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFromPath(root).Load(); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(logs.String(), "Several workspace configurations"); got != 1 {
		t.Fatalf("conflict warning emitted %d times, want 1:\n%s", got, logs.String())
	}
}

func TestLoad_RootWithoutSelfLineStaysFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, gws.WorkspacesFileName), "sub | folder\n")
	ws, err := NewFromPath(root).Load()
	if err != nil {
		t.Fatal(err)
	}
	if ws.IsGitRepository() {
		t.Fatalf("root without '.' line must not be git-backed, got %+v", ws.Remotes)
	}
}

func TestSaveWorkspace_KeepsSelfLine(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, gws.WorkspacesFileName)
	writeFile(t, file, ". | git@example.com:org/root.git\nsub | folder\n")
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	ws, err := NewFromPath(root).Recursive(false).Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.SaveWorkspace(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 || lines[0] != ". | git@example.com:org/root.git" || lines[1] != "sub | folder" {
		t.Fatalf("saved file = %q", string(data))
	}
}

func TestFilterIgnoredProjects(t *testing.T) {
	root := t.TempDir()
	projects := []*gws2.Project{
		gws2.NewProject(root, "ignore-me", nil),
		gws2.NewProject(root, "keep-me", nil),
	}
	cases := []struct {
		name     string
		patterns []string
		want     int
	}{
		{"no patterns", nil, 2},
		{"matches one", []string{"ignore-me"}, 1},
		{"matches all", []string{"-me$"}, 0},
		{"invalid regexp is skipped", []string{"[invalid"}, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := filterIgnoredProjects(projects, gws2.CompileIgnoreRules(c.patterns)); len(got) != c.want {
				t.Fatalf("len = %d, want %d", len(got), c.want)
			}
		})
	}
}

func TestReadIgnorePatterns(t *testing.T) {
	root := t.TempDir()
	patterns, err := readIgnorePatterns(root)
	if err != nil || len(patterns) != 0 {
		t.Fatalf("missing file: patterns=%v err=%v", patterns, err)
	}

	writeFile(t, filepath.Join(root, gws.IgnoreFileName), "# comment\npattern1\n\npattern2\n# another comment\npattern3\n")
	patterns, err = readIgnorePatterns(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(patterns) != 3 || patterns[0] != "pattern1" || patterns[2] != "pattern3" {
		t.Fatalf("patterns = %v", patterns)
	}
}
