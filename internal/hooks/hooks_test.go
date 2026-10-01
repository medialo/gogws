package hooks

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/medialo/gogws/internal/engine"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/hookfile"
)

func writeHook(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func TestHookTypesAreKnownByHookfile(t *testing.T) {
	for _, h := range []HookType{HookPreInit, HookPostInit, HookPreUpdate, HookPostUpdate, HookPreClone, HookPostClone, HookPreFetch, HookPostFetch, HookPreFF, HookPostFF, HookPreCheck, HookPostCheck} {
		if !slices.Contains(hookfile.Hooks, string(h)) {
			t.Errorf("%s is missing from hookfile.Hooks", h)
		}
	}
}

func TestFindHook_DetectsExtensionAndIgnoresUnknownSuffixes(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	hooksDir := filepath.Join(root, ".gws", "hooks")
	writeHook(t, filepath.Join(hooksDir, "post-ff.sample"), "x")
	writeHook(t, filepath.Join(hooksDir, "post-ff.sh.bak"), "x")
	writeHook(t, filepath.Join(hooksDir, "post-ff.sh"), "x")

	got, err := findHook(HookPostFF, root)
	if err != nil || got == nil || filepath.Base(got.Path) != "post-ff.sh" || got.Origin != OriginLocal {
		t.Fatalf("got %+v, %v; want local post-ff.sh", got, err)
	}
}

func TestFindHook_CurrentOSDirWins(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	hooksDir := filepath.Join(root, ".gws", "hooks")
	writeHook(t, filepath.Join(hooksDir, "post-ff.sh"), "x")
	writeHook(t, filepath.Join(hooksDir, runtime.GOOS, "post-ff.js"), "x")

	got, err := findHook(HookPostFF, root)
	if err != nil || got == nil || filepath.Base(filepath.Dir(got.Path)) != runtime.GOOS {
		t.Fatalf("got %+v, %v; want the %s hook", got, err, runtime.GOOS)
	}
}

func TestFindHook_ConflictIsAnError(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	hooksDir := filepath.Join(root, ".gws", "hooks")
	writeHook(t, filepath.Join(hooksDir, "post-ff"), "x")
	writeHook(t, filepath.Join(hooksDir, "post-ff.sh"), "x")

	_, err := findHook(HookPostFF, root)
	var conflict *hookfile.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("err = %v; want ConflictError", err)
	}
	if err := Run(HookPostFF, root, Context{Command: "ff", WorkspaceRoot: root}); !errors.As(err, &conflict) {
		t.Fatalf("Run err = %v; want ConflictError", err)
	}
}

func TestPrepareProjectHooks_ConflictFailsBeforeAnyPrompt(t *testing.T) {
	isolateHome(t)
	SetTrustMode(TrustModeAsk)
	promptInput = strings.NewReader("")
	t.Cleanup(func() { promptInput = os.Stdin })

	root := t.TempDir()
	api := newProject(root, "api")
	web := newProject(root, "web")
	ws := &gws2.Workspace{Entry: gws2.Entry{AbsolutePath: root}, Projects: []*gws2.Project{web, api}}
	hooksDir := filepath.Join(root, ".gws", "hooks")
	writeHook(t, filepath.Join(hooksDir, "web.post-ff.sh"), "x")
	writeHook(t, filepath.Join(hooksDir, runtime.GOOS, "api.post-ff.sh"), "x")
	writeHook(t, filepath.Join(hooksDir, runtime.GOOS, "api.post-ff.js"), "x")

	_, err := PrepareProjectHooks(ws, ProjectHooksOptions{Command: "ff", Pre: HookPreFF, Post: HookPostFF})
	var conflict *hookfile.ConflictError
	if !errors.As(err, &conflict) || conflict.Target != "api" {
		t.Fatalf("err = %v; want api conflict", err)
	}
	if state, _, _ := HookTrustState(filepath.Join(hooksDir, "web.post-ff.sh")); state != TrustStateNew {
		t.Fatalf("web hook must not have been prompted/trusted, state = %v", state)
	}
}

func newProject(dir, name string) *gws2.Project {
	return &gws2.Project{GitRepository: gws2.GitRepository{Entry: gws2.Entry{AbsolutePath: filepath.Join(dir, name), Name: name}}}
}

func TestPrepareProjectHooks_IsNotRecursive(t *testing.T) {
	isolateHome(t)
	SetTrustMode(TrustModeAll)
	t.Cleanup(func() { SetTrustMode(TrustModeAsk) })

	root := t.TempDir()
	subDir := filepath.Join(root, "sub")
	rootAPI := newProject(root, "api")
	subAPI := newProject(subDir, "api")
	subWeb := newProject(subDir, "web")
	sub := &gws2.Workspace{Entry: gws2.Entry{AbsolutePath: subDir, Name: "sub"}, Projects: []*gws2.Project{subAPI, subWeb}}
	ws := &gws2.Workspace{Entry: gws2.Entry{AbsolutePath: root}, Projects: []*gws2.Project{rootAPI}, Children: []*gws2.Workspace{sub}}

	writeHook(t, filepath.Join(root, ".gws", "hooks", "api.post-ff.sh"), "x")
	writeHook(t, filepath.Join(subDir, ".gws", "hooks", "web.post-ff"), "x")

	set, err := PrepareProjectHooks(ws, ProjectHooksOptions{Command: "ff", Pre: HookPreFF, Post: HookPostFF})
	if err != nil {
		t.Fatal(err)
	}

	if hooks := set.byProject[rootAPI.GetPath()].post; len(hooks) != 1 || hooks[0].Origin != OriginProject {
		t.Fatalf("root api: got %+v, want one project hook", hooks)
	}
	if hooks := set.byProject[subAPI.GetPath()].post; len(hooks) != 0 {
		t.Fatalf("sub api must not inherit the parent hook, got %+v", hooks)
	}
	if hooks := set.byProject[subWeb.GetPath()].post; len(hooks) != 1 {
		t.Fatalf("sub web: got %+v, want its own hook", hooks)
	}
}

func TestPrepareProjectHooks_SkipModeDropsUntrusted(t *testing.T) {
	isolateHome(t)
	SetTrustMode(TrustModeSkip)
	t.Cleanup(func() { SetTrustMode(TrustModeAsk) })

	root := t.TempDir()
	api := newProject(root, "api")
	ws := &gws2.Workspace{Entry: gws2.Entry{AbsolutePath: root}, Projects: []*gws2.Project{api}}
	writeHook(t, filepath.Join(root, ".gws", "hooks", "api.post-ff.sh"), "x")

	set, err := PrepareProjectHooks(ws, ProjectHooksOptions{Command: "ff", Pre: HookPreFF, Post: HookPostFF})
	if err != nil {
		t.Fatal(err)
	}
	if hooks := set.byProject[api.GetPath()].post; len(hooks) != 0 {
		t.Fatalf("untrusted hook must be skipped, got %+v", hooks)
	}
}

func TestHookTrustState_PerFileAndContent(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	hook := filepath.Join(dir, "post-ff.sh")
	writeHook(t, hook, "echo v1")

	state, sum, err := HookTrustState(hook)
	if err != nil || state != TrustStateNew {
		t.Fatalf("state = %v, err = %v; want new", state, err)
	}
	if err := TrustHookFile(hook, sum); err != nil {
		t.Fatal(err)
	}
	if state, _, _ := HookTrustState(hook); state != TrustStateTrusted {
		t.Fatalf("state = %v; want trusted", state)
	}

	writeHook(t, hook, "echo v2")
	if state, _, _ := HookTrustState(hook); state != TrustStateModified {
		t.Fatalf("state = %v; want modified", state)
	}

	other := filepath.Join(dir, "pre-ff.sh")
	writeHook(t, other, "echo v1")
	if state, _, _ := HookTrustState(other); state != TrustStateNew {
		t.Fatalf("new file in the same workspace: state = %v; want new", state)
	}
}

func TestApprove_AskPromptTrustRecordsFile(t *testing.T) {
	isolateHome(t)
	SetTrustMode(TrustModeAsk)
	promptInput = strings.NewReader("t\n")
	t.Cleanup(func() { promptInput = os.Stdin })

	dir := t.TempDir()
	path := filepath.Join(dir, "post-ff.sh")
	writeHook(t, path, "echo hi")

	ok, err := approve(&HookInfo{Name: HookPostFF, Path: path, Origin: OriginLocal}, dir)
	if err != nil || !ok {
		t.Fatalf("approve = %v, %v; want true", ok, err)
	}
	if state, _, _ := HookTrustState(path); state != TrustStateTrusted {
		t.Fatalf("state = %v; want trusted after [t]", state)
	}
}

func TestApprove_ConsecutivePromptsShareInput(t *testing.T) {
	isolateHome(t)
	SetTrustMode(TrustModeAsk)
	promptInput = strings.NewReader("t\nt\n")
	t.Cleanup(func() { promptInput = os.Stdin })

	dir := t.TempDir()
	for _, name := range []string{"api.post-ff.sh", "post-ff.sh"} {
		path := filepath.Join(dir, name)
		writeHook(t, path, "echo "+name)
		ok, err := approve(&HookInfo{Name: HookPostFF, Path: path, Origin: OriginLocal}, dir)
		if err != nil || !ok {
			t.Fatalf("%s: approve = %v, %v; want true", name, ok, err)
		}
		if state, _, _ := HookTrustState(path); state != TrustStateTrusted {
			t.Fatalf("%s: state = %v; want trusted", name, state)
		}
	}
}

func fakeLookPath(t *testing.T, available ...string) {
	t.Helper()
	set := map[string]bool{}
	for _, a := range available {
		set[a] = true
	}
	original := lookPath
	lookPath = func(name string) (string, error) {
		if set[name] {
			return "/fake/bin/" + name, nil
		}
		return "", errors.New("not found")
	}
	t.Cleanup(func() { lookPath = original })
}

func TestResolveInterpreter_ByExtension(t *testing.T) {
	fakeLookPath(t, "bash", "powershell", "python")
	dir := t.TempDir()

	cases := map[string][]string{
		"a.sh":  {"/fake/bin/bash", filepath.ToSlash(filepath.Join(dir, "a.sh"))},
		"a.ps1": {"/fake/bin/powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(dir, "a.ps1")},
		"a.py":  {"/fake/bin/python", filepath.Join(dir, "a.py")},
	}
	for name, want := range cases {
		got, err := resolveInterpreter(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}
}

func TestResolveInterpreter_MissingInterpreter(t *testing.T) {
	fakeLookPath(t)
	_, err := resolveInterpreter(filepath.Join(t.TempDir(), "a.js"))
	if err == nil || !strings.Contains(err.Error(), "node not found") {
		t.Fatalf("err = %v; want node not found", err)
	}
}

func TestResolveInterpreter_ShebangOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("shebang parsing is only used on windows")
	}
	fakeLookPath(t, "bash")
	dir := t.TempDir()

	withShebang := filepath.Join(dir, "post-ff")
	writeHook(t, withShebang, "#!/usr/bin/env bash\necho hi\n")
	got, err := resolveInterpreter(withShebang)
	if err != nil || got[0] != "/fake/bin/bash" {
		t.Fatalf("got %v, %v; want bash", got, err)
	}

	noShebang := filepath.Join(dir, "pre-ff")
	writeHook(t, noShebang, "echo hi\n")
	if _, err := resolveInterpreter(noShebang); err == nil {
		t.Fatal("want an error for a hook without extension nor shebang")
	}
}

func scriptHook(t *testing.T, dir, base, body string, exit int) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, base+".cmd")
		writeHook(t, path, "@echo off\r\n"+body+"\r\nexit /b "+strconv.Itoa(exit)+"\r\n")
		return path
	}
	path := filepath.Join(dir, base+".sh")
	writeHook(t, path, body+"\nexit "+strconv.Itoa(exit)+"\n")
	return path
}

type recorded struct {
	kind engine.EventType
	log  string
}

func TestAround_RunsPostHookInJobWithPhaseAndEnv(t *testing.T) {
	root := t.TempDir()
	api := newProject(root, "api")
	if err := os.MkdirAll(api.GetPath(), 0o755); err != nil {
		t.Fatal(err)
	}
	ws := &gws2.Workspace{Entry: gws2.Entry{AbsolutePath: root}}

	echo := "echo %GOGWS_PROJECT_NAME%"
	if runtime.GOOS != "windows" {
		echo = "echo $GOGWS_PROJECT_NAME"
	}
	post := &HookInfo{Name: HookPostFF, Path: scriptHook(t, root, "api.post-ff", echo, 0), Origin: OriginProject}
	set := &ProjectHooks{command: "ff", workspaceRoot: root, byProject: map[string]*projectHooks{
		api.GetPath(): {owner: ws, post: []*HookInfo{post}},
	}}

	var events []recorded
	notify := func(kind engine.EventType, log string) { events = append(events, recorded{kind, log}) }

	if err := set.Around(context.Background(), api, notify, func() error { return nil }); err != nil {
		t.Fatalf("Around: %v", err)
	}

	if len(events) == 0 || events[0].kind != engine.EventJobPhase || !strings.Contains(events[0].log, "post-ff") {
		t.Fatalf("first event = %+v; want post-ff phase", events)
	}
	found := false
	for _, e := range events {
		if e.kind == engine.EventJobLog && strings.TrimSpace(e.log) == "api" {
			found = true
		}
	}
	if !found {
		t.Fatalf("hook output with GOGWS_PROJECT_NAME not forwarded: %+v", events)
	}
	if last := events[len(events)-1]; last.kind != engine.EventJobPhase || last.log != "" {
		t.Fatalf("last event = %+v; want phase reset", last)
	}
}

func TestAround_HookExitCodeFailsJob(t *testing.T) {
	root := t.TempDir()
	api := newProject(root, "api")
	ws := &gws2.Workspace{Entry: gws2.Entry{AbsolutePath: root}}
	post := &HookInfo{Name: HookPostFF, Path: scriptHook(t, root, "api.post-ff", "echo boom", 3), Origin: OriginProject}
	set := &ProjectHooks{command: "ff", workspaceRoot: root, byProject: map[string]*projectHooks{
		api.GetPath(): {owner: ws, post: []*HookInfo{post}},
	}}

	err := set.Around(context.Background(), api, func(engine.EventType, string) {}, func() error { return nil })
	if err == nil || !strings.Contains(err.Error(), "post-ff hook failed") {
		t.Fatalf("err = %v; want post-ff hook failed", err)
	}
}

func TestAround_PostHookSkippedWhenOperationFails(t *testing.T) {
	root := t.TempDir()
	api := newProject(root, "api")
	ws := &gws2.Workspace{Entry: gws2.Entry{AbsolutePath: root}}
	post := &HookInfo{Name: HookPostFF, Path: filepath.Join(root, "missing.sh"), Origin: OriginProject}
	set := &ProjectHooks{command: "ff", workspaceRoot: root, byProject: map[string]*projectHooks{
		api.GetPath(): {owner: ws, post: []*HookInfo{post}},
	}}

	opErr := errors.New("ff failed")
	err := set.Around(context.Background(), api, func(engine.EventType, string) {}, func() error { return opErr })
	if !errors.Is(err, opErr) {
		t.Fatalf("err = %v; want the operation error untouched", err)
	}
}
