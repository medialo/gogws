package gws2

import (
	"path/filepath"
	"testing"
)

func TestIgnoreRules(t *testing.T) {
	rules := CompileIgnoreRules([]string{"ignore-me", "[invalid", "^/tmp/"})

	if rules.IsEmpty() {
		t.Fatal("rules should not be empty")
	}
	if !rules.Match("/root/ignore-me") || !rules.Match("/tmp/x") {
		t.Fatal("expected matches")
	}
	if rules.Match("/root/keep-me") {
		t.Fatal("unexpected match")
	}
	if !CompileIgnoreRules(nil).IsEmpty() || !CompileIgnoreRules([]string{"[invalid"}).IsEmpty() {
		t.Fatal("rules without valid pattern should be empty")
	}
}

func TestRemoveProjects_Batch(t *testing.T) {
	root := t.TempDir()
	ws := NewRootWorkspace(root)
	ws.AttachIndex(NewIndex())
	child := NewFolderWorkspace(root, "child")
	ws.AddWorkspace(child)
	for _, name := range []string{"a", "b", "c"} {
		ws.AddProject(NewProject(root, name, nil))
	}
	child.AddProject(NewProject(child.GetPath(), "d", nil))

	owners := ws.RemoveProjects(filepath.Join(root, "a"), filepath.Join(root, "missing"), filepath.Join(child.GetPath(), "d"))

	if owners[0] != ws || owners[1] != nil || owners[2] != child {
		t.Fatalf("owners = %v", owners)
	}
	if len(ws.Projects) != 2 || len(child.Projects) != 0 {
		t.Fatalf("remaining: root %d, child %d", len(ws.Projects), len(child.Projects))
	}
	if ws.Index().Has(filepath.Join(root, "a")) || !ws.Index().Has(filepath.Join(root, "b")) {
		t.Fatal("index not rebuilt")
	}
}

func TestConfigFiles_Lookup(t *testing.T) {
	files := NewConfigFiles("/ws", []string{"workspaces.yaml", ".projects.gws"}, []string{"projects.gws"})

	if path, ok := files.Lookup(false, "workspaces.yaml"); !ok || path != filepath.Join("/ws", "workspaces.yaml") {
		t.Fatalf("root lookup = %q, %v", path, ok)
	}
	if path, ok := files.Lookup(true, "projects.gws"); !ok || path != filepath.Join("/ws", ConfigDirName, "projects.gws") {
		t.Fatalf("config dir lookup = %q, %v", path, ok)
	}
	if _, ok := files.Lookup(true, "workspaces.yaml"); ok {
		t.Fatal("file from root found in config dir")
	}
}
