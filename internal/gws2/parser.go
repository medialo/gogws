package gws2

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func parseProjectsFile(root string) ([]*Project, error) {
	_, location := getProjectsConfigFileLocation(root)
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

		project, err := parseProjectLine(root, line)
		if err != nil {
			return nil, fmt.Errorf("error parsing line %d: %w", lineNum, err)
		}

		project.id = lineNum
		projects = append(projects, project)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading projects file: %w", err)
	}

	ignorePatterns, err := parseIgnoreFile(root)
	if err == nil && len(ignorePatterns) > 0 {
		projects = filterIgnoredProjects(projects, ignorePatterns)
	}

	return projects, nil
}

func parseWorkspacesFile(root string) ([]*Workspace, error) {
	_, location := getWorkspacesConfigFileLocation(root)
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

		ws, err := parseWorkspaceLine(line)
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

func parseProjectLine(root string, line string) (*Project, error) {
	parts := strings.Split(line, "|")
	if len(parts) < 2 {
		return &Project{}, fmt.Errorf("invalid format: expected 'path | url [name] [| url2 name2 ...]'")
	}

	path := filepath.Join(root, strings.TrimSpace(parts[0]))
	if path == "" {
		return &Project{}, fmt.Errorf("empty project path")
	}

	project := &Project{
		GitRepository: GitRepository{
			Path:          path,
			Remotes:       make([]*Remote, 0),
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

func parseWorkspaceLine(line string) (*Workspace, error) {
	parts := strings.Split(line, "|")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid format: expected 'path | url [name]'")
	}

	path := strings.TrimSpace(parts[0])
	if path == "" {
		return nil, fmt.Errorf("empty workspace path")
	}

	remotePart := strings.TrimSpace(parts[1])
	if remotePart == "" {
		return nil, fmt.Errorf("empty remote URL for workspace %s", path)
	}

	remote, err := parseRemote(remotePart, 0)
	if err != nil {
		return nil, err
	}

	return &Workspace{
		GitRepository: GitRepository{
			Path:          path,
			Remotes:       []*Remote{remote},
			Name:          filepath.Base(path),
			Type:          RepositoryTypeWorkspace,
			FolderExists:  false,
			gitRepository: true, // read from gws config file so is expected to be a git repository
		},
		Projects: []*Project{},
		Children: []*Workspace{},
	}, nil
}

func parseRemote(remotePart string, index int) (*Remote, error) {
	fields := strings.Fields(remotePart)
	if len(fields) == 0 {
		return &Remote{}, fmt.Errorf("empty remote definition")
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

	return &Remote{
		Name: name,
		URL:  url,
	}, nil
}

func filterIgnoredProjects(projects []*Project, patterns []string) []*Project {
	if len(patterns) == 0 {
		return projects
	}

	regexps := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if re, err := regexp.Compile(pattern); err == nil {
			regexps = append(regexps, re)
		}
	}

	filtered := make([]*Project, 0, len(projects))
	for _, project := range projects {
		ignored := false
		for _, re := range regexps {
			if re.MatchString(project.Path) {
				ignored = true
				break
			}
		}
		if !ignored {
			filtered = append(filtered, project)
		}
	}

	return filtered
}
