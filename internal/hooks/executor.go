package hooks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/medialo/gogws/internal/ui/styles"
)

type HookType string

const (
	HookPreInit    HookType = "pre-init"
	HookPostInit   HookType = "post-init"
	HookPreUpdate  HookType = "pre-update"
	HookPostUpdate HookType = "post-update"
	HookPreClone   HookType = "pre-clone"
	HookPostClone  HookType = "post-clone"
	HookPreFetch   HookType = "pre-fetch"
	HookPostFetch  HookType = "post-fetch"
	HookPreFF      HookType = "pre-ff"
	HookPostFF     HookType = "post-ff"
	HookPreCheck   HookType = "pre-check"
	HookPostCheck  HookType = "post-check"
)

type HookOrigin string

const (
	OriginGlobal  HookOrigin = "global"
	OriginLocal   HookOrigin = "local"
	OriginProject HookOrigin = "project"
)

type HookInfo struct {
	Name    HookType
	Path    string
	Origin  HookOrigin
	Trusted bool
}

func (h *HookInfo) Label() string {
	if h.Trusted {
		return fmt.Sprintf("hook:%s:trusted", h.Origin)
	}
	return fmt.Sprintf("hook:%s", h.Origin)
}

type Context struct {
	Command       string
	WorkspaceRoot string
	ProjectDir    string
	ProjectName   string
	Projects      []string
	Data          map[string]interface{}
}

var globalTrustMode TrustMode = TrustModeAsk

func SetTrustMode(mode TrustMode) {
	globalTrustMode = mode
}

func GetTrustMode() TrustMode {
	return globalTrustMode
}

func hookEnv(ctx Context, hook *HookInfo) []string {
	env := []string{
		"GOGWS_COMMAND=" + ctx.Command,
		"GOGWS_WORKSPACE_DIR=" + ctx.WorkspaceRoot,
		"GOGWS_WORKSPACE=" + ctx.WorkspaceRoot,
		"GOGWS_HOOK_NAME=" + string(hook.Name),
		"GOGWS_HOOK_ORIGIN=" + string(hook.Origin),
	}
	if ctx.ProjectDir != "" {
		env = append(env, "GOGWS_PROJECT_DIR="+ctx.ProjectDir, "GOGWS_PROJECT_NAME="+ctx.ProjectName)
	}
	if len(ctx.Projects) > 0 {
		env = append(env, "GOGWS_PROJECTS="+strings.Join(ctx.Projects, "\n"))
	}
	keys := make([]string, 0, len(ctx.Data))
	for k := range ctx.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		env = append(env, fmt.Sprintf("GOGWS_%s=%v", strings.ToUpper(k), ctx.Data[k]))
	}
	return env
}

func exitDescription(err error) string {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return fmt.Sprintf("exit %d", exitErr.ExitCode())
	}
	return err.Error()
}

func executeHook(hook *HookInfo, workspaceRoot string, ctx Context) error {
	if hook == nil {
		return nil
	}

	ok, err := approve(hook, workspaceRoot)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	s := styles.Get()
	lipgloss.Println(s.Info.Render(fmt.Sprintf("[%s] %s", hook.Label(), hook.Name)) + " " + s.Muted.Render(hook.Path))

	cmd, err := buildCommand(context.Background(), hook, workspaceRoot, hookEnv(ctx, hook))
	if err != nil {
		lipgloss.Println(s.Error.Render(fmt.Sprintf("%s %s: %v", s.IconError, hook.Name, err)))
		return err
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start := time.Now()
	err = cmd.Run()
	elapsed := time.Since(start).Round(time.Millisecond)
	if err != nil {
		lipgloss.Println(s.Error.Render(fmt.Sprintf("%s %s %s", s.IconError, hook.Name, exitDescription(err))) + " " + s.Muted.Render(elapsed.String()))
		return err
	}
	lipgloss.Println(s.Success.Render(fmt.Sprintf("%s %s", s.IconSuccess, hook.Name)) + " " + s.Muted.Render(elapsed.String()))
	return nil
}

func Run(hookName HookType, workspaceRoot string, ctx Context) error {
	hook, err := findHook(hookName, workspaceRoot)
	if err != nil {
		return err
	}
	if hook == nil {
		return nil
	}
	return executeHook(hook, workspaceRoot, ctx)
}

func PreInit(workspaceRoot string) error {
	return Run(HookPreInit, workspaceRoot, Context{
		Command:       "init",
		WorkspaceRoot: workspaceRoot,
	})
}

func PostInit(workspaceRoot string, projects []string) error {
	return Run(HookPostInit, workspaceRoot, Context{
		Command:       "init",
		WorkspaceRoot: workspaceRoot,
		Projects:      projects,
	})
}

func PreUpdate(workspaceRoot string) error {
	return Run(HookPreUpdate, workspaceRoot, Context{
		Command:       "update",
		WorkspaceRoot: workspaceRoot,
	})
}

func PostUpdate(workspaceRoot string, cloned []string) error {
	return Run(HookPostUpdate, workspaceRoot, Context{
		Command:       "update",
		WorkspaceRoot: workspaceRoot,
		Projects:      cloned,
	})
}

func PreFetch(workspaceRoot string) error {
	return Run(HookPreFetch, workspaceRoot, Context{
		Command:       "fetch",
		WorkspaceRoot: workspaceRoot,
	})
}

func PostFetch(workspaceRoot string, fetched int) error {
	return Run(HookPostFetch, workspaceRoot, Context{
		Command:       "fetch",
		WorkspaceRoot: workspaceRoot,
		Data: map[string]interface{}{
			"fetched": fetched,
		},
	})
}

func PreFF(workspaceRoot string) error {
	return Run(HookPreFF, workspaceRoot, Context{
		Command:       "ff",
		WorkspaceRoot: workspaceRoot,
	})
}

func PostFF(workspaceRoot string, pulled int) error {
	return Run(HookPostFF, workspaceRoot, Context{
		Command:       "ff",
		WorkspaceRoot: workspaceRoot,
		Data: map[string]interface{}{
			"pulled": pulled,
		},
	})
}

func PreCheck(workspaceRoot string) error {
	return Run(HookPreCheck, workspaceRoot, Context{
		Command:       "check",
		WorkspaceRoot: workspaceRoot,
	})
}

func PostCheck(workspaceRoot string, unknown []string) error {
	return Run(HookPostCheck, workspaceRoot, Context{
		Command:       "check",
		WorkspaceRoot: workspaceRoot,
		Projects:      unknown,
	})
}
