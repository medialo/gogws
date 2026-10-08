package interactive

import (
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

const EnvNoInteractive = "GOGWS_NO_INTERACTIVE"

var (
	mu             sync.Mutex
	disabled       bool
	resolved       bool
	enabled        bool
	promptResolved bool
	canPrompt      bool
	isTTY          = func(f *os.File) bool { return term.IsTerminal(int(f.Fd())) }
)

func Disable() {
	mu.Lock()
	defer mu.Unlock()
	disabled = true
	resolved = false
	promptResolved = false
}

func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()
	if !resolved {
		enabled = detect()
		resolved = true
	}
	return enabled
}

func CanPrompt() bool {
	mu.Lock()
	defer mu.Unlock()
	if !promptResolved {
		canPrompt = !forcedOff() && isTTY(os.Stdin)
		promptResolved = true
	}
	return canPrompt
}

func detect() bool {
	return !forcedOff() && isTTY(os.Stdout) && isTTY(os.Stdin)
}

func forcedOff() bool {
	return disabled || isTruthy(os.Getenv(EnvNoInteractive)) || isTruthy(os.Getenv("CI"))
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func reset() {
	mu.Lock()
	defer mu.Unlock()
	disabled = false
	resolved = false
	promptResolved = false
}
