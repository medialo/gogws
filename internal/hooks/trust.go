package hooks

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/medialo/gogws/internal/config"
)

type TrustMode string

const (
	TrustModeAsk  TrustMode = "ask"
	TrustModeAll  TrustMode = "all"
	TrustModeSkip TrustMode = "skip"
)

type TrustResult int

const (
	TrustResultRun TrustResult = iota
	TrustResultSkip
	TrustResultRunAndTrust
)

func IsWorkspaceTrusted(workspacePath string) bool {
	cfg, err := config.LoadUserConfig()
	if err != nil {
		return false
	}

	absPath, err := filepath.Abs(workspacePath)
	if err != nil {
		return false
	}

	for _, pattern := range cfg.TrustedWorkspaces {
		if matchPattern(pattern, absPath) {
			return true
		}
	}

	return false
}

func matchPattern(pattern, path string) bool {
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(path, prefix)
	}

	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		if !strings.HasPrefix(path, prefix) {
			return false
		}
		remaining := strings.TrimPrefix(path, prefix)
		remaining = strings.TrimPrefix(remaining, "/")
		return !strings.Contains(remaining, "/")
	}

	if strings.Contains(pattern, "*") {
		matched, _ := filepath.Match(pattern, path)
		return matched
	}

	return pattern == path
}

func PromptTrust(hookName, hookPath, workspacePath string) TrustResult {
	fmt.Printf("\n[hook:local] Hook '%s' found at: %s\n", hookName, hookPath)
	fmt.Printf("Workspace: %s\n", workspacePath)
	fmt.Println("This workspace is not in your trusted list.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  [r] Run this hook")
	fmt.Println("  [s] Skip this hook")
	fmt.Println("  [t] Run and add workspace to trusted list")
	fmt.Print("Choose [r/s/t]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return TrustResultSkip
	}

	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "r", "run":
		return TrustResultRun
	case "t", "trust":
		return TrustResultRunAndTrust
	default:
		return TrustResultSkip
	}
}

func AddToTrusted(workspacePath string) error {
	absPath, err := filepath.Abs(workspacePath)
	if err != nil {
		return err
	}
	return config.AddTrustedWorkspace(absPath)
}

func ParseTrustMode(s string) TrustMode {
	switch strings.ToLower(s) {
	case "all":
		return TrustModeAll
	case "skip":
		return TrustModeSkip
	default:
		return TrustModeAsk
	}
}
