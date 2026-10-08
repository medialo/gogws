package gws

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gws2"
)

func parseProjectsFile(rootPath, path string) ([]*gws2.Project, error) {
	var projects []*gws2.Project
	err := scanLines(path, func(lineNum int, line string) error {
		project, err := parseProjectLine(rootPath, line)
		if err != nil {
			return fmt.Errorf("error parsing line %d: %w", lineNum, err)
		}
		projects = append(projects, project)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func parseWorkspacesFile(rootPath, path string) ([]*gws2.Workspace, error) {
	var workspaces []*gws2.Workspace
	err := scanLines(path, func(lineNum int, line string) error {
		ws, err := parseWorkspaceLine(rootPath, line)
		if err != nil {
			return fmt.Errorf("error parsing line %d in %s: %w", lineNum, filepath.Base(path), err)
		}
		workspaces = append(workspaces, ws)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return workspaces, nil
}

func scanLines(path string, handle func(lineNum int, line string) error) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

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

		if err := handle(lineNum, line); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading %s: %w", path, err)
	}
	return nil
}

func parseProjectLine(rootPath string, line string) (*gws2.Project, error) {
	parts := strings.Split(line, "|")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid format: expected 'path | url [name] [| url2 name2 ...]'")
	}

	relPath := strings.TrimSpace(parts[0])
	path := filepath.Join(rootPath, relPath)
	if path == "" {
		return nil, fmt.Errorf("empty project path")
	}

	remotes := make([]*git.Remote, 0)
	for i := 1; i < len(parts); i++ {
		remotePart := strings.TrimSpace(parts[i])
		if remotePart == "" {
			continue
		}

		remote, err := parseRemote(remotePart, i-1)
		if err != nil {
			return nil, err
		}
		remotes = append(remotes, remote)
	}

	if len(remotes) == 0 {
		return nil, fmt.Errorf("no remotes defined for project %s", path)
	}

	return gws2.NewProject(rootPath, relPath, remotes), nil
}

func parseWorkspaceLine(rootPath string, line string) (*gws2.Workspace, error) {
	parts := strings.Split(line, "|")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid format: expected 'path | url [name]'")
	}

	relPath := strings.TrimSpace(parts[0])
	path := filepath.Join(rootPath, relPath)
	if path == "" {
		return nil, fmt.Errorf("empty workspace path")
	}

	remotePart := strings.TrimSpace(parts[1])
	if remotePart == "" {
		return nil, fmt.Errorf("empty remote URL or type for workspace %s", path)
	}

	if remotePart == "folder" {
		return gws2.NewFolderWorkspace(rootPath, relPath), nil
	}

	remote, err := parseRemote(remotePart, 0)
	if err != nil {
		return nil, err
	}
	return gws2.NewChildWorkspace(rootPath, relPath, []*git.Remote{remote}), nil
}

func parseRemote(remotePart string, index int) (*git.Remote, error) {
	fields := strings.Fields(remotePart)
	if len(fields) == 0 {
		return nil, fmt.Errorf("empty remote definition")
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
