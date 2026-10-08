package loader

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/gws2/format/gws"
)

func LoadIgnorePatterns(rootPath string) ([]string, error) {
	ignorePath := filepath.Join(rootPath, gws.IgnoreFileName)
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

func filterIgnoredProjects(projects []*gws2.Project, patterns []string) []*gws2.Project {
	if len(patterns) == 0 {
		return projects
	}

	filtered := make([]*gws2.Project, 0, len(projects))
	for _, project := range projects {
		if !gws2.IsPathIgnored(project.AbsolutePath, patterns) {
			filtered = append(filtered, project)
		}
	}

	return filtered
}
