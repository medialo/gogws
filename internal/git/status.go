package git

import (
	"bufio"
	"bytes"
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

	parsePorcelainStatus(output, status)

	branches, err := getBranches(repoPath)
	if err == nil {
		status.Branches = branches
	}

	return status
}

func parsePorcelainStatus(output []byte, status *RepositoryStatus) {
	uncommitted, untracked := 0, 0
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "# branch.oid "):
			status.Oid = strings.TrimPrefix(line, "# branch.oid ")
		case strings.HasPrefix(line, "# branch.head "):
			status.Branch = strings.TrimPrefix(line, "# branch.head ")
		case strings.HasPrefix(line, "# branch.upstream"):
			status.HasRemote = true
		case strings.HasPrefix(line, "# branch.ab "):
			fields := strings.Fields(strings.TrimPrefix(line, "# branch.ab "))
			if len(fields) == 2 {
				status.Ahead, _ = strconv.Atoi(strings.TrimPrefix(fields[0], "+"))
				status.Behind, _ = strconv.Atoi(strings.TrimPrefix(fields[1], "-"))
			}
		case strings.HasPrefix(line, "?"):
			untracked++
		case len(line) > 0 && line[0] != '#':
			uncommitted++
		}
	}
	status.Uncommitted = uncommitted
	status.Untracked = untracked
	status.Clean = uncommitted == 0 && untracked == 0
}

func IsRepository(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

func getBranches(repoPath string) ([]BranchStatus, error) {
	cmd := exec.Command("git", "for-each-ref",
		"--format=%(refname:short)|%(upstream:short)|%(HEAD)|%(upstream:track,nobracket)",
		"refs/heads/")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseBranches(output), nil
}

func parseBranches(output []byte) []BranchStatus {
	var branches []BranchStatus
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 3 {
			continue
		}

		branch := BranchStatus{
			Name:      parts[0],
			Upstream:  parts[1],
			IsCurrent: parts[2] == "*",
		}
		if len(parts) == 4 && branch.Upstream != "" {
			branch.Ahead, branch.Behind = parseTrack(parts[3])
		}

		branches = append(branches, branch)
	}

	return branches
}

func parseTrack(track string) (ahead, behind int) {
	for part := range strings.SplitSeq(track, ",") {
		fields := strings.Fields(part)
		if len(fields) != 2 {
			continue
		}
		n, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		switch fields[0] {
		case "ahead":
			ahead = n
		case "behind":
			behind = n
		}
	}
	return ahead, behind
}
