package git

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
)

type Remote struct {
	Name string
	URL  string
}

func defaultRunner(ctx context.Context, cmd *exec.Cmd) error {
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

func Clone(ctx context.Context, targetPath string, remotes []Remote, run func(context.Context, *exec.Cmd) error) error {
	if len(remotes) == 0 {
		return fmt.Errorf("no remotes defined")
	}

	if run == nil {
		run = defaultRunner
	}

	primaryRemote := remotes[0]

	cmd := exec.CommandContext(ctx, "git", "clone", "--progress", primaryRemote.URL, targetPath)
	if err := run(ctx, cmd); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	for i := 1; i < len(remotes); i++ {
		remote := remotes[i]
		cmd := exec.CommandContext(ctx, "git", "remote", "add", remote.Name, remote.URL)
		cmd.Dir = targetPath
		if err := run(ctx, cmd); err != nil {
			return fmt.Errorf("failed to add remote %s: %w", remote.Name, err)
		}
	}

	return nil
}

func CloneWorkspace(ctx context.Context, workspaceRoot string, path string, remotes []Remote, run func(context.Context, *exec.Cmd) error) error {
	targetPath := filepath.Join(workspaceRoot, path)
	return Clone(ctx, targetPath, remotes, run)
}
