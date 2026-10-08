package git

import (
	"slices"
	"sync"
	"testing"
)

func withGlobalSSHCommand(t *testing.T, value string) {
	t.Helper()
	previous := lookupGlobalSSH
	lookupGlobalSSH = func() string { return value }
	globalSSHCommandOnce = sync.Once{}
	t.Cleanup(func() {
		lookupGlobalSSH = previous
		globalSSHCommandOnce = sync.Once{}
	})
}

func TestUnattendedEnv(t *testing.T) {
	cases := []struct {
		name      string
		sshEnv    string
		globalSSH string
		wantSSH   bool
	}{
		{"default", "", "", true},
		{"user GIT_SSH_COMMAND kept", "ssh -i key", "", false},
		{"global core.sshCommand kept", "", "ssh -i key", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("GIT_SSH_COMMAND", c.sshEnv)
			t.Setenv("GIT_SSH", "")
			withGlobalSSHCommand(t, c.globalSSH)

			env := UnattendedEnv()
			if !slices.Contains(env, "GIT_TERMINAL_PROMPT=0") || !slices.Contains(env, "GCM_INTERACTIVE=never") {
				t.Fatalf("env = %v", env)
			}
			if got := slices.Contains(env, "GIT_SSH_COMMAND="+batchSSHCommand); got != c.wantSSH {
				t.Fatalf("batch ssh set = %v, want %v (env %v)", got, c.wantSSH, env)
			}
		})
	}
}

func TestAuthenticationHint(t *testing.T) {
	if AuthenticationHint("fatal: could not read Username for 'https://x': terminal prompts disabled") == "" {
		t.Fatal("expected hint for disabled prompts")
	}
	if AuthenticationHint("git@x: Permission denied (publickey).") == "" {
		t.Fatal("expected hint for ssh key")
	}
	if AuthenticationHint("fatal: repository not found") != "" {
		t.Fatal("unexpected hint")
	}
}
