package git

import (
	"fmt"
	"os/exec"
)

type Cmd exec.Cmd

// Run is a custom méthod to run git command without calling CombinedOutput
// to allow git command to return a *exec.Cmd
func (cmd *Cmd) Run() error {
	if output, err := (*exec.Cmd)(cmd).CombinedOutput(); err != nil {
		return fmt.Errorf("command %q failed: %s", cmd.Args, string(output))
	}
	return nil
}

func (cmd *Cmd) AsCmd() *exec.Cmd {
	return (*exec.Cmd)(cmd)
}
