package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/gws"
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
	OriginGlobal HookOrigin = "global"
	OriginLocal  HookOrigin = "local"
)

type HookInfo struct {
	Name   HookType
	Path   string
	Origin HookOrigin
}

type Context struct {
	Command       string
	WorkspaceRoot string
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

func findHook(hookName HookType, workspaceRoot string) *HookInfo {
	localHooksDir := filepath.Join(workspaceRoot, gws.ConfigDirName, gws.HooksDirName)
	localHookPath := filepath.Join(localHooksDir, string(hookName))

	if info, err := os.Stat(localHookPath); err == nil && !info.IsDir() {
		return &HookInfo{
			Name:   hookName,
			Path:   localHookPath,
			Origin: OriginLocal,
		}
	}

	globalHooksDir, err := config.GetUserHooksDir()
	if err == nil {
		globalHookPath := filepath.Join(globalHooksDir, string(hookName))
		if info, err := os.Stat(globalHookPath); err == nil && !info.IsDir() {
			return &HookInfo{
				Name:   hookName,
				Path:   globalHookPath,
				Origin: OriginGlobal,
			}
		}
	}

	return nil
}

func executeHook(hook *HookInfo, workspaceRoot string, ctx Context) error {
	if hook == nil {
		return nil
	}

	if hook.Origin == OriginLocal {
		if !IsWorkspaceTrusted(workspaceRoot) {
			switch globalTrustMode {
			case TrustModeSkip:
				fmt.Printf("[hook:%s] Skipping untrusted hook: %s\n", hook.Origin, hook.Name)
				return nil
			case TrustModeAll:
				fmt.Printf("[hook:%s] Running hook (trust-mode=all): %s\n", hook.Origin, hook.Name)
			case TrustModeAsk:
				result := PromptTrust(string(hook.Name), hook.Path, workspaceRoot)
				switch result {
				case TrustResultSkip:
					fmt.Printf("[hook:%s] Skipped by user: %s\n", hook.Origin, hook.Name)
					return nil
				case TrustResultRunAndTrust:
					if err := AddToTrusted(workspaceRoot); err != nil {
						fmt.Printf("Warning: failed to add workspace to trusted list: %v\n", err)
					} else {
						fmt.Printf("Workspace added to trusted list\n")
					}
				}
			}
		} else {
			fmt.Printf("[hook:%s:trusted] %s\n", hook.Origin, hook.Name)
		}
	} else {
		fmt.Printf("[hook:%s] %s\n", hook.Origin, hook.Name)
	}

	cmd := exec.Command(hook.Path)
	cmd.Dir = workspaceRoot
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GOGWS_COMMAND=%s", ctx.Command),
		fmt.Sprintf("GOGWS_WORKSPACE=%s", ctx.WorkspaceRoot),
		fmt.Sprintf("GOGWS_HOOK_NAME=%s", hook.Name),
		fmt.Sprintf("GOGWS_HOOK_ORIGIN=%s", hook.Origin),
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func Run(hookName HookType, workspaceRoot string, ctx Context) error {
	hook := findHook(hookName, workspaceRoot)
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

func PreClone(workspaceRoot string, repoPath string) error {
	return Run(HookPreClone, workspaceRoot, Context{
		Command:       "clone",
		WorkspaceRoot: workspaceRoot,
		Projects:      []string{repoPath},
	})
}

func PostClone(workspaceRoot string, repoPath string, success bool) error {
	return Run(HookPostClone, workspaceRoot, Context{
		Command:       "clone",
		WorkspaceRoot: workspaceRoot,
		Projects:      []string{repoPath},
		Data: map[string]interface{}{
			"success": success,
		},
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
