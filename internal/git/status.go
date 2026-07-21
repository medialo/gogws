package git

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func GetStatus(repoPath string) *RepositoryStatus {
	slog.Debug("Getting status for repository", "path", repoPath, "context", "GIT_OP")
	status := &RepositoryStatus{
		Path:   repoPath,
		Exists: false,
	}

	if _, err := os.Stat(filepath.Join(repoPath, ".git")); os.IsNotExist(err) {
		status.Error = err
		return status
	}
	status.Exists = true

	cmd := exec.Command("git", "status", "--porcelain=v2", "--branch")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		status.Error = err
		return status
	}

	uncommitted, untracked := 0, 0
	for _, line := range strings.Split(string(output), "\n") {
		switch {
		case strings.HasPrefix(line, "# branch.oid "):
			status.Oid = strings.TrimPrefix(line, "# branch.oid ")
		case strings.HasPrefix(line, "# branch.head "):
			status.Branch = strings.TrimPrefix(line, "# branch.head ")
		case strings.HasPrefix(line, "# branch.upstream"):
			status.HasRemote = true
		case strings.HasPrefix(line, "# branch.ab "):
			fmt.Sscanf(line, "# branch.ab +%d -%d", &status.Ahead, &status.Behind)
		case strings.HasPrefix(line, "?"):
			untracked++
		case len(line) > 0 && line[0] != '#':
			uncommitted++
		}
	}
	status.Uncommitted = uncommitted
	status.Untracked = untracked
	status.Clean = uncommitted == 0 && untracked == 0

	branches, err := getBranches(repoPath)
	if err == nil {
		status.Branches = branches
	}

	return status
}

func getBranches(repoPath string) ([]BranchStatus, error) {
	cmd := exec.Command("git", "for-each-ref",
		"--format=%(refname:short)|%(upstream:short)|%(HEAD)",
		"refs/heads/")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var branches []BranchStatus
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}

		branchName := parts[0]
		upstream := parts[1]
		isCurrent := parts[2] == "*"

		branch := BranchStatus{
			Name:      branchName,
			IsCurrent: isCurrent,
			Upstream:  upstream,
		}

		if upstream != "" {
			ahead, behind := getAheadBehind(repoPath, branchName, upstream)
			branch.Ahead = ahead
			branch.Behind = behind
		}

		branches = append(branches, branch)
	}

	return branches, nil
}

func getAheadBehind(repoPath, branch, upstream string) (ahead, behind int) {
	cmd := exec.Command("git", "rev-list", "--left-right", "--count",
		fmt.Sprintf("%s...%s", branch, upstream))
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return 0, 0
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) == 2 {
		ahead, _ = strconv.Atoi(parts[0])
		behind, _ = strconv.Atoi(parts[1])
	}

	return ahead, behind
}
