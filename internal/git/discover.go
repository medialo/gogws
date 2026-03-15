package git

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type DiscoveredRepo struct {
	Path    string
	Remotes []Remote
}

func DiscoverRepositories(rootPath string, maxDepth int) ([]DiscoveredRepo, error) {
	slog.Debug("Starting repository discovery", "rootPath", rootPath, "maxDepth", maxDepth)

	var repos []DiscoveredRepo

	if maxDepth < 1 {
		maxDepth = 1
	}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}

		depth := len(strings.Split(relPath, string(os.PathSeparator)))
		if relPath != "." && depth > maxDepth {
			return filepath.SkipDir
		}

		gitDir := filepath.Join(path, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			if relPath == "." {
				return nil
			}

			remotes, err := getRemotesExec(path)
			if err != nil || len(remotes) == 0 {
				return filepath.SkipDir
			}

			discovered := DiscoveredRepo{
				Path:    relPath,
				Remotes: remotes,
			}

			repos = append(repos, discovered)
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to discover repositories: %w", err)
	}

	slog.Debug("Completed repository discovery", "count", len(repos))
	return repos, nil
}

func getRemotesExec(repoPath string) ([]Remote, error) {
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	remoteMap := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			name := parts[0]
			url := parts[1]
			if _, exists := remoteMap[name]; !exists {
				remoteMap[name] = url
			}
		}
	}

	var remotes []Remote
	for name, url := range remoteMap {
		remotes = append(remotes, Remote{Name: name, URL: url})
	}

	return remotes, nil
}

func FindUnknownRepositories(rootPath string, knownPaths []string) ([]string, error) {
	allRepos, err := DiscoverRepositories(rootPath, 10)
	if err != nil {
		return nil, err
	}

	known := make(map[string]bool)
	for _, path := range knownPaths {
		known[path] = true
	}

	var unknown []string
	for _, repo := range allRepos {
		if !known[repo.Path] {
			unknown = append(unknown, repo.Path)
		}
	}

	return unknown, nil
}
