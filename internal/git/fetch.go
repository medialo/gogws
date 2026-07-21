package git

import (
	"log/slog"
	"os/exec"
)

func Fetch(repoPath string) *Cmd {
	slog.Debug("Fetching repository", "path", repoPath, "verbose", "VVV", "context", "GIT_OP")

	cmd := exec.Command("git", "fetch", "--all", "--progress")
	cmd.Dir = repoPath
	return (*Cmd)(cmd)
}

func Pull(repoPath string) *Cmd {
	//slog.Log(context.Background(), log.DebugLevel2, "Pulling repository", "path", repoPath, "verbose", "VVV", "context", "GIT_OP")
	cmd := exec.Command("git", "pull", "--ff-only", "--progress")
	cmd.Dir = repoPath
	return (*Cmd)(cmd)
}
