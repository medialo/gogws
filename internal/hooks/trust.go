package hooks

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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

type TrustState int

const (
	TrustStateTrusted TrustState = iota
	TrustStateNew
	TrustStateModified
)

func (s TrustState) String() string {
	switch s {
	case TrustStateTrusted:
		return "trusted"
	case TrustStateModified:
		return "modified since it was trusted"
	default:
		return "new, never trusted"
	}
}

var (
	promptInput  io.Reader = os.Stdin
	promptReader *bufio.Reader
	promptSource io.Reader
)

func readPromptLine() (string, error) {
	if promptReader == nil || promptSource != promptInput {
		promptReader = bufio.NewReader(promptInput)
		promptSource = promptInput
	}
	return promptReader.ReadString('\n')
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func HookTrustState(hookPath string) (TrustState, string, error) {
	absPath, err := filepath.Abs(hookPath)
	if err != nil {
		return TrustStateNew, "", err
	}
	sum, err := fileSHA256(absPath)
	if err != nil {
		return TrustStateNew, "", err
	}
	trusted, ok := config.FindTrustedHook(absPath)
	switch {
	case !ok:
		return TrustStateNew, sum, nil
	case trusted.SHA256 != sum:
		return TrustStateModified, sum, nil
	default:
		return TrustStateTrusted, sum, nil
	}
}

func TrustHookFile(hookPath, sum string) error {
	absPath, err := filepath.Abs(hookPath)
	if err != nil {
		return err
	}
	return config.TrustHook(absPath, sum)
}

func approve(hook *HookInfo, workspaceRoot string) (bool, error) {
	if hook.Origin == OriginGlobal {
		return true, nil
	}

	state, sum, err := HookTrustState(hook.Path)
	if err != nil {
		return false, err
	}
	if state == TrustStateTrusted {
		hook.Trusted = true
		return true, nil
	}

	switch globalTrustMode {
	case TrustModeSkip:
		fmt.Printf("[hook:%s] Skipping untrusted hook (%s): %s\n", hook.Origin, state, hook.Path)
		return false, nil
	case TrustModeAll:
		return true, nil
	}

	switch PromptTrust(hook, workspaceRoot, state) {
	case TrustResultSkip:
		fmt.Printf("[hook:%s] Skipped by user: %s\n", hook.Origin, hook.Path)
		return false, nil
	case TrustResultRunAndTrust:
		if err := TrustHookFile(hook.Path, sum); err != nil {
			fmt.Printf("Warning: failed to trust hook file: %v\n", err)
		} else {
			hook.Trusted = true
		}
	}
	return true, nil
}

func PromptTrust(hook *HookInfo, workspacePath string, state TrustState) TrustResult {
	fmt.Printf("\n[hook:%s] Hook '%s' found at: %s\n", hook.Origin, hook.Name, hook.Path)
	fmt.Printf("Workspace: %s\n", workspacePath)
	fmt.Printf("This hook file is %s.\n", state)
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  [r] Run this hook once")
	fmt.Println("  [s] Skip this hook")
	fmt.Println("  [t] Run and trust this file (asked again if it changes)")
	fmt.Print("Choose [r/s/t]: ")

	input, err := readPromptLine()
	if err != nil && input == "" {
		return TrustResultSkip
	}

	switch strings.TrimSpace(strings.ToLower(input)) {
	case "r", "run":
		return TrustResultRun
	case "t", "trust":
		return TrustResultRunAndTrust
	default:
		return TrustResultSkip
	}
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
