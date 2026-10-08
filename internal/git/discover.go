package git

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type DiscoveredRepo struct {
	Path    string
	Remotes []*Remote
}

const discoverParallel = 5

func DiscoverRepositories(rootPath string, maxDepth int) ([]DiscoveredRepo, error) {
	slog.Debug("Starting repository discovery", "rootPath", rootPath, "maxDepth", maxDepth)

	candidates, err := walkRepositories(rootPath, maxDepth)
	if err != nil {
		return nil, err
	}

	found := make([]*DiscoveredRepo, len(candidates))
	sem := make(chan struct{}, discoverParallel)
	var wg sync.WaitGroup
	for i, rel := range candidates {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			remotes, err := getRemotesExec(filepath.Join(rootPath, rel))
			if err == nil && len(remotes) > 0 {
				found[i] = &DiscoveredRepo{Path: rel, Remotes: remotes}
			}
		}()
	}
	wg.Wait()

	repos := make([]DiscoveredRepo, 0, len(found))
	for _, repo := range found {
		if repo != nil {
			repos = append(repos, *repo)
		}
	}

	slog.Debug("Completed repository discovery", "count", len(repos))
	return repos, nil
}

func DiscoverRepositoryPaths(rootPath string, maxDepth int) ([]string, error) {
	candidates, err := walkRepositories(rootPath, maxDepth)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(candidates))
	for _, rel := range candidates {
		if hasRemote(filepath.Join(rootPath, rel)) {
			paths = append(paths, rel)
		}
	}
	return paths, nil
}

func walkRepositories(rootPath string, maxDepth int) ([]string, error) {
	if maxDepth < 1 {
		maxDepth = 1
	}

	var candidates []string
	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		depth := strings.Count(relPath, string(os.PathSeparator)) + 1
		if depth > maxDepth {
			return filepath.SkipDir
		}

		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			candidates = append(candidates, relPath)
			return filepath.SkipDir
		}

		if depth == maxDepth {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to discover repositories: %w", err)
	}
	return candidates, nil
}

func hasRemote(repoPath string) bool {
	gitDir := filepath.Join(repoPath, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		remotes, err := getRemotesExec(repoPath)
		return err == nil && len(remotes) > 0
	}
	data, err := os.ReadFile(filepath.Join(gitDir, "config"))
	if err != nil {
		return false
	}
	return bytes.Contains(data, []byte("[remote \""))
}

func getRemotesExec(repoPath string) ([]*Remote, error) {
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

	var remotes []*Remote
	for name, url := range remoteMap {
		remotes = append(remotes, &Remote{Name: name, URL: url})
	}

	sort.Slice(remotes, func(i, j int) bool {
		if (remotes[i].Name == "origin") != (remotes[j].Name == "origin") {
			return remotes[i].Name == "origin"
		}
		return remotes[i].Name < remotes[j].Name
	})

	return remotes, nil
}

func FindUnknownRepositories(rootPath string, knownPaths []string) ([]string, error) {
	allRepos, err := DiscoverRepositoryPaths(rootPath, 0)
	if err != nil {
		return nil, err
	}

	// DiscoverRepositories reports paths relative to rootPath, while known
	// paths may be absolute (e.g. gws2 project paths) or relative — normalize
	// both to absolute paths anchored at rootPath before comparing.
	known := make(map[string]bool)
	for _, path := range knownPaths {
		known[normalizeRepoPath(rootPath, path)] = true
	}

	var unknown []string
	for _, repo := range allRepos {
		if !known[normalizeRepoPath(rootPath, repo)] {
			unknown = append(unknown, repo)
		}
	}

	return unknown, nil
}

func DiscoverRepository(rootPath, path string) (*DiscoveredRepo, error) {
	if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
		return nil, nil
	}

	relPath, err := filepath.Rel(rootPath, path)
	if err != nil {
		return nil, err
	}
	if relPath == "." {
		return nil, nil
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(os.PathSeparator)) {
		return nil, fmt.Errorf("%s is outside workspace %s", path, rootPath)
	}

	remotes, err := getRemotesExec(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read remotes of %s: %w", path, err)
	}

	return &DiscoveredRepo{Path: relPath, Remotes: remotes}, nil
}

func normalizeRepoPath(rootPath, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(rootPath, path))
}
