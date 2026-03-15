package git

import (
	"log/slog"
	"os/exec"
)

func Fetch(repoPath string) *Cmd {
	slog.Debug("Fetching repository", "path", repoPath)

	cmd := exec.Command("git", "fetch", "--all")
	cmd.Dir = repoPath
	return (*Cmd)(cmd)
}

func Pull(repoPath string) *Cmd {
	cmd := exec.Command("git", "pull", "--ff-only")
	cmd.Dir = repoPath
	return (*Cmd)(cmd)
}
