package hooks

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/hookfile"
)

func isHookFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func localHooksDir(workspaceDir string) string {
	return filepath.Join(workspaceDir, gws2.ConfigDirName, gws2.HooksDirName)
}

func findHook(hookName HookType, workspaceRoot string) (*HookInfo, error) {
	return findHookWith(hookfile.Scan, hookName, workspaceRoot)
}

func findHookWith(scan hookfile.Scanner, hookName HookType, workspaceRoot string) (*HookInfo, error) {
	localDir := localHooksDir(workspaceRoot)
	file, err := hookfile.Resolve(scan, hookfile.SearchDirs(localDir, runtime.GOOS), "", string(hookName))
	if err != nil {
		return nil, err
	}
	if file != nil {
		return &HookInfo{Name: hookName, Path: file.Path, Origin: OriginLocal}, nil
	}

	globalDir, err := config.GetUserHooksDir()
	if err == nil {
		file, err := hookfile.Resolve(scan, hookfile.SearchDirs(globalDir, runtime.GOOS), "", string(hookName))
		if err != nil {
			return nil, err
		}
		if file != nil {
			return &HookInfo{Name: hookName, Path: file.Path, Origin: OriginGlobal}, nil
		}
	}

	slog.Debug("No hook found", "hook", hookName, "local", localDir, "global", globalDir, "os", runtime.GOOS)
	return nil, nil
}

func findProjectHook(scan hookfile.Scanner, owner *gws2.Workspace, project *gws2.Project, hookName HookType) (*HookInfo, error) {
	dirs := hookfile.SearchDirs(localHooksDir(owner.GetPath()), runtime.GOOS)
	file, err := hookfile.Resolve(scan, dirs, project.GetName(), string(hookName))
	if err != nil || file == nil {
		return nil, err
	}
	return &HookInfo{Name: hookName, Path: file.Path, Origin: OriginProject}, nil
}
