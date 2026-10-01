package hooks

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/medialo/gogws/internal/engine"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/hookfile"
)

type projectHooks struct {
	owner *gws2.Workspace
	pre   []*HookInfo
	post  []*HookInfo
}

type ProjectHooks struct {
	command       string
	workspaceRoot string
	byProject     map[string]*projectHooks
}

type ProjectHooksOptions struct {
	Command          string
	Pre              HookType
	Post             HookType
	PerRepoWorkspace bool
}

type ownedProject struct {
	project *gws2.Project
	owner   *gws2.Workspace
}

func ownedProjects(root *gws2.Workspace) []ownedProject {
	var owned []ownedProject
	for _, ws := range append([]*gws2.Workspace{root}, root.FlattenWorkspaces()...) {
		for _, p := range ws.Projects {
			owned = append(owned, ownedProject{project: p, owner: ws})
		}
	}
	return owned
}

func approveAll(found []*HookInfo, workspaceRoot string, approved map[*HookInfo]bool) ([]*HookInfo, error) {
	kept := make([]*HookInfo, 0, len(found))
	for _, h := range found {
		if h == nil {
			continue
		}
		ok, seen := approved[h]
		if !seen {
			var err error
			ok, err = approve(h, workspaceRoot)
			if err != nil {
				return nil, err
			}
			approved[h] = ok
		}
		if ok {
			kept = append(kept, h)
		}
	}
	return kept, nil
}

func PrepareProjectHooks(root *gws2.Workspace, opts ProjectHooksOptions) (*ProjectHooks, error) {
	workspaceRoot := root.GetPath()
	set := &ProjectHooks{
		command:       opts.Command,
		workspaceRoot: workspaceRoot,
		byProject:     make(map[string]*projectHooks),
	}

	scan := hookfile.CachedScanner()

	var sharedPre, sharedPost *HookInfo
	if opts.PerRepoWorkspace {
		var err error
		if sharedPre, err = findHookWith(scan, opts.Pre, workspaceRoot); err != nil {
			return nil, err
		}
		if sharedPost, err = findHookWith(scan, opts.Post, workspaceRoot); err != nil {
			return nil, err
		}
	}

	type discovered struct {
		ownedProject
		pre, post *HookInfo
	}
	var found []discovered
	for _, op := range ownedProjects(root) {
		pre, err := findProjectHook(scan, op.owner, op.project, opts.Pre)
		if err != nil {
			return nil, err
		}
		post, err := findProjectHook(scan, op.owner, op.project, opts.Post)
		if err != nil {
			return nil, err
		}
		found = append(found, discovered{ownedProject: op, pre: pre, post: post})
	}

	approved := make(map[*HookInfo]bool)
	for _, d := range found {
		pre, err := approveAll([]*HookInfo{sharedPre, d.pre}, d.owner.GetPath(), approved)
		if err != nil {
			return nil, err
		}
		post, err := approveAll([]*HookInfo{sharedPost, d.post}, d.owner.GetPath(), approved)
		if err != nil {
			return nil, err
		}
		set.byProject[d.project.GetPath()] = &projectHooks{owner: d.owner, pre: pre, post: post}
	}
	return set, nil
}

func (s *ProjectHooks) Around(ctx context.Context, project *gws2.Project, notify engine.Notify, op func() error) error {
	if s == nil {
		return op()
	}
	hooks := s.byProject[project.GetPath()]
	if hooks == nil {
		return op()
	}

	for _, h := range hooks.pre {
		if err := s.runInJob(ctx, h, hooks.owner, project, notify); err != nil {
			return err
		}
	}
	if err := op(); err != nil {
		return err
	}
	for _, h := range hooks.post {
		if err := s.runInJob(ctx, h, hooks.owner, project, notify); err != nil {
			return err
		}
	}
	return nil
}

func (s *ProjectHooks) runInJob(ctx context.Context, hook *HookInfo, owner *gws2.Workspace, project *gws2.Project, notify engine.Notify) error {
	dir := project.GetPath()
	if !isDir(dir) {
		dir = owner.GetPath()
	}

	env := hookEnv(Context{
		Command:       s.command,
		WorkspaceRoot: s.workspaceRoot,
		ProjectDir:    project.GetPath(),
		ProjectName:   project.GetName(),
	}, hook)

	notify(engine.EventJobPhase, fmt.Sprintf("hook %s (%s)", hook.Name, filepath.Base(hook.Path)))
	defer notify(engine.EventJobPhase, "")

	cmd, err := buildCommand(ctx, hook, dir, env)
	if err != nil {
		return fmt.Errorf("%s hook failed: %w", hook.Name, err)
	}
	if err := engine.Wrap(cmd).Run(ctx, notify); err != nil {
		return fmt.Errorf("%s hook failed: %w", hook.Name, err)
	}
	return nil
}
