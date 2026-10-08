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

func LoadIgnoreRules(rootPath string) (gws2.IgnoreRules, error) {
	patterns, err := readIgnorePatterns(rootPath)
	if err != nil {
		return gws2.IgnoreRules{}, err
	}
	return gws2.CompileIgnoreRules(patterns), nil
}

func readIgnorePatterns(rootPath string) ([]string, error) {
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

func filterIgnoredProjects(projects []*gws2.Project, rules gws2.IgnoreRules) []*gws2.Project {
	if rules.IsEmpty() {
		return projects
	}

	filtered := make([]*gws2.Project, 0, len(projects))
	for _, project := range projects {
		if !rules.Match(project.AbsolutePath) {
			filtered = append(filtered, project)
		}
	}

	return filtered
}
