package git

import (
	"os"
	"os/exec"
	"strings"
	"sync"
)

const batchSSHCommand = "ssh -o BatchMode=yes"

var (
	globalSSHCommandOnce sync.Once
	globalSSHCommand     string
	lookupGlobalSSH      = readGlobalSSHCommand // var used to allow test to inject fake case
)

func UnattendedEnv() []string {
	env := []string{"GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=never"}
	if os.Getenv("GIT_SSH_COMMAND") == "" && os.Getenv("GIT_SSH") == "" && globalCoreSSHCommand() == "" {
		env = append(env, "GIT_SSH_COMMAND="+batchSSHCommand)
	}
	return env
}

func WithUnattendedEnv(cmd *exec.Cmd) {
	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}
	cmd.Env = append(cmd.Env, UnattendedEnv()...)
}

func AuthenticationHint(output string) string {
	if strings.Contains(output, "terminal prompts disabled") ||
		strings.Contains(output, "Permission denied (publickey") ||
		strings.Contains(output, "Host key verification failed") {
		return "authentication required: configure a git credential helper or ssh-agent"
	}
	return ""
}

func globalCoreSSHCommand() string {
	globalSSHCommandOnce.Do(func() {
		globalSSHCommand = lookupGlobalSSH()
	})
	return globalSSHCommand
}

func readGlobalSSHCommand() string {
	output, err := exec.Command("git", "config", "--global", "--get", "core.sshCommand").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
