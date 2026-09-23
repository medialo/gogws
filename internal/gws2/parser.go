package gws2

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/medialo/gogws/internal/git"
)

func parseProjectsFile(rootPath string) ([]*Project, error) {
	_, location := getProjectsConfigFileLocation(rootPath)
	if location == nil {
		return nil, fmt.Errorf("no projects file found")
	}

	projectsPath := location.Path

	file, err := os.Open(projectsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open projects file: %w", err)
	}
	defer file.Close()

	var projects []*Project
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		project, err := parseProjectLine(rootPath, line)
		if err != nil {
			return nil, fmt.Errorf("error parsing line %d: %w", lineNum, err)
		}

		project.id = lineNum
		projects = append(projects, project)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading projects file: %w", err)
	}

	ignorePatterns, err := parseIgnoreFile(rootPath)
	if err == nil && len(ignorePatterns) > 0 {
		projects = filterIgnoredProjects(projects, ignorePatterns)
	}

	return projects, nil
}

func parseWorkspacesFile(rootPath string) ([]*Workspace, error) {
	_, location := getWorkspacesConfigFileLocation(rootPath)
	if location == nil {
		return []*Workspace{}, nil
	}

	workspacesPath := location.Path

	file, err := os.Open(workspacesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Workspace{}, nil
		}
		return nil, fmt.Errorf("failed to open workspaces file: %w", err)
	}
	defer file.Close()

	var workspaces []*Workspace
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		ws, err := parseWorkspaceLine(rootPath, line)
		if err != nil {
			return nil, fmt.Errorf("error parsing line %d in %s: %w", lineNum, WorkspacesFileName, err)
		}
		ws.id = lineNum
		workspaces = append(workspaces, ws)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading workspaces file: %w", err)
	}

	return workspaces, nil
}

func parseIgnoreFile(root string) ([]string, error) {
	ignorePath := filepath.Join(root, IgnoreFileName)
	file, err := os.Open(ignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to open ignore file: %w", err)
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading ignore file: %w", err)
	}

	return patterns, nil
}

func parseProjectLine(rootPath string, line string) (*Project, error) {
	parts := strings.Split(line, "|")
	if len(parts) < 2 {
		return &Project{}, fmt.Errorf("invalid format: expected 'path | url [name] [| url2 name2 ...]'")
	}

	path := filepath.Join(rootPath, strings.TrimSpace(parts[0]))
	if path == "" {
		return &Project{}, fmt.Errorf("empty project path")
	}

	project := &Project{
		GitRepository: GitRepository{
			Path:          path,
			Remotes:       make([]*git.Remote, 0),
			Name:          filepath.Base(path),
			Type:          RepositoryTypeProject,
			gitRepository: true,
			FolderExists:  false, // no tested here
		},
	}

	for i := 1; i < len(parts); i++ {
		remotePart := strings.TrimSpace(parts[i])
		if remotePart == "" {
			continue
		}

		remote, err := parseRemote(remotePart, i-1)
		if err != nil {
			return &Project{}, err
		}
		project.Remotes = append(project.Remotes, remote)
	}

	if len(project.Remotes) == 0 {
		return &Project{}, fmt.Errorf("no remotes defined for project %s", path)
	}

	return project, nil
}

func parseWorkspaceLine(rootPath string, line string) (*Workspace, error) {
	parts := strings.Split(line, "|")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid format: expected 'path | url [name]'")
	}

	path := filepath.Join(rootPath, strings.TrimSpace(parts[0]))
	if path == "" {
		return nil, fmt.Errorf("empty workspace path")
	}

	var _type RepositoryType
	remotePart := strings.TrimSpace(parts[1])
	if remotePart == "" {
		return nil, fmt.Errorf("empty remote URL or type for workspace %s", path)
	}

	var _remote *git.Remote

	if "folder" == remotePart {
		_type = RepositoryTypeFolder
		_remote = nil
	} else {
		_type = RepositoryTypeWorkspace
		var err error
		_remote, err = parseRemote(remotePart, 0)
		if err != nil {
			return nil, err
		}
	}

	return &Workspace{
		GitRepository: GitRepository{
			Path:          path,
			Remotes:       []*git.Remote{_remote},
			Name:          filepath.Base(path),
			Type:          _type,
			FolderExists:  false,
			gitRepository: true, // read from gws config file so is expected to be a git repository
		},
		Projects: []*Project{},
		Children: []*Workspace{},
	}, nil
}

func parseRemote(remotePart string, index int) (*git.Remote, error) {
	fields := strings.Fields(remotePart)
	if len(fields) == 0 {
		return &git.Remote{}, fmt.Errorf("empty remote definition")
	}

	url := fields[0]
	name := "origin"

	if index == 0 && len(fields) > 1 {
		name = fields[1]
	} else if index == 1 {
		name = "upstream"
		if len(fields) > 1 {
			name = fields[1]
		}
	} else if len(fields) > 1 {
		name = fields[1]
	}

	return &git.Remote{
		Name: name,
		URL:  url,
	}, nil
}

func filterIgnoredProjects(projects []*Project, patterns []string) []*Project {
	if len(patterns) == 0 {
		return projects
	}

	filtered := make([]*Project, 0, len(projects))
	for _, project := range projects {
		if !IsPathIgnored(project.Path, patterns) {
			filtered = append(filtered, project)
		}
	}

	return filtered
}

// LoadIgnorePatterns returns the ignore patterns declared in the workspace's
// ignore file (see IgnoreFileName), so callers can distinguish paths that
// were deliberately excluded from paths that are genuinely unknown.
func LoadIgnorePatterns(rootPath string) ([]string, error) {
	return parseIgnoreFile(rootPath)
}

// IsPathIgnored reports whether path matches any of the given ignore patterns.
// Invalid patterns are skipped, matching the behavior of the ignore file parser.
func IsPathIgnored(path string, patterns []string) bool {
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if re.MatchString(path) {
			return true
		}
	}
	return false
}
