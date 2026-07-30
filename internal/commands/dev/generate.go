package dev

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/medialo/gogws/internal/gitignore"
	"github.com/medialo/gogws/internal/gws"

	"github.com/spf13/cobra"
)

var (
	numProjects       int
	numWorkspaces     int
	maxDepth          int
	baseURL           string
	initRepos         bool
	outputDir         string
	prefix            string
	generateGitIgnore bool
)

type projectInfo struct {
	Name       string
	Path       string
	RemoteURL  string
	Depth      int
	ParentPath string
}

type workspaceInfo struct {
	Name       string
	Path       string
	RemoteURL  string
	Depth      int
	ParentPath string
	Projects   []projectInfo
	Workspaces []workspaceInfo
}

func NewGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a test workspace structure",
		Long: `Generate a recursive workspace structure with projects and sub-workspaces
for testing and development purposes.

The generated structure includes:
- .gws/projects.gws files with project definitions
- .gws/workspaces.gws files with workspace definitions (if depth > 0)
- Optionally, real git repositories with README.md files (--init-repos)

Names follow the pattern: <prefix>-<path>-<type><index>
Example: test-w1-w2-p3 (project 3 in workspace 2 of workspace 1)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate()
		},
	}

	cmd.Flags().IntVar(&numProjects, "projects", 5, "number of projects per workspace")
	cmd.Flags().IntVar(&numWorkspaces, "workspaces", 2, "number of sub-workspaces per level")
	cmd.Flags().IntVar(&maxDepth, "depth", 1, "maximum recursion depth for workspaces")
	cmd.Flags().StringVar(&baseURL, "base-url", "git@github.com:test", "base URL for git remotes")
	cmd.Flags().BoolVar(&initRepos, "init-repos", false, "create actual git repositories with README.md")
	cmd.Flags().StringVar(&outputDir, "output", ".", "output directory")
	cmd.Flags().StringVar(&prefix, "prefix", "test", "prefix for generated names")
	cmd.Flags().BoolVar(&generateGitIgnore, "gitignore", true, "generate .gitignore files for workspaces (use --gitignore=false to disable)")

	return cmd
}

func runGenerate() error {
	absOutput, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory: %w", err)
	}

	fmt.Printf("Generating test workspace structure in: %s\n", absOutput)
	fmt.Printf("  Projects per workspace: %d\n", numProjects)
	fmt.Printf("  Workspaces per level: %d\n", numWorkspaces)
	fmt.Printf("  Max depth: %d\n", maxDepth)
	fmt.Printf("  Base URL: %s\n", baseURL)
	fmt.Printf("  Init repos: %v\n", initRepos)
	fmt.Printf("  Generate .gitignore: %v\n", generateGitIgnore)
	fmt.Println()

	wsInfo := generateWorkspace(absOutput, prefix, "", 0)

	if initRepos {
		if err := createRootReadme(absOutput, wsInfo); err != nil {
			return fmt.Errorf("failed to create root README: %w", err)
		}
	}

	fmt.Println()
	fmt.Printf("Generation complete!\n")
	printStats(wsInfo)

	return nil
}

func generateWorkspace(dir, pfx, parentPath string, depth int) workspaceInfo {
	gwsDir := filepath.Join(dir, gws.ConfigDirName)
	if err := os.MkdirAll(gwsDir, 0755); err != nil {
		fmt.Printf("Error creating %s directory: %v\n", gws.ConfigDirName, err)
		return workspaceInfo{}
	}

	if generateGitIgnore {
		if err := gitignore.CreateGitignore(dir); err != nil {
			fmt.Printf("Error creating .gitignore: %v\n", err)
		}
	}

	wsInfo := workspaceInfo{
		Name:       buildName(pfx, parentPath, ""),
		Path:       parentPath,
		Depth:      depth,
		ParentPath: parentPath,
		Projects:   []projectInfo{},
		Workspaces: []workspaceInfo{},
	}

	for i := 1; i <= numProjects; i++ {
		name := buildName(pfx, parentPath, fmt.Sprintf("p%d", i))
		remoteURL := fmt.Sprintf("%s/%s.git", baseURL, name)

		proj := projectInfo{
			Name:       name,
			Path:       name,
			RemoteURL:  remoteURL,
			Depth:      depth,
			ParentPath: parentPath,
		}
		wsInfo.Projects = append(wsInfo.Projects, proj)

		if initRepos {
			projDir := filepath.Join(dir, name)
			if err := createProjectRepo(projDir, proj); err != nil {
				fmt.Printf("Error creating project repo %s: %v\n", name, err)
			} else {
				fmt.Printf("  Created project: %s\n", name)
			}
		}
	}

	if err := writeProjectsFile(gwsDir, wsInfo.Projects); err != nil {
		fmt.Printf("Error writing projects.gws: %v\n", err)
	}

	if depth < maxDepth {
		for i := 1; i <= numWorkspaces; i++ {
			name := buildName(pfx, parentPath, fmt.Sprintf("w%d", i))
			remoteURL := fmt.Sprintf("%s/%s.git", baseURL, name)

			var newPath string
			if parentPath == "" {
				newPath = fmt.Sprintf("w%d", i)
			} else {
				newPath = fmt.Sprintf("%s-w%d", parentPath, i)
			}

			childDir := filepath.Join(dir, name)
			if err := os.MkdirAll(childDir, 0755); err != nil {
				fmt.Printf("Error creating workspace directory %s: %v\n", name, err)
				continue
			}

			childWs := generateWorkspace(childDir, pfx, newPath, depth+1)
			childWs.Name = name
			childWs.RemoteURL = remoteURL

			wsInfo.Workspaces = append(wsInfo.Workspaces, childWs)

			if initRepos {
				if err := createWorkspaceRepo(childDir, childWs); err != nil {
					fmt.Printf("Error creating workspace repo %s: %v\n", name, err)
				} else {
					fmt.Printf("  Created workspace: %s\n", name)
				}
			}
		}

		if err := writeWorkspacesFile(gwsDir, wsInfo.Workspaces); err != nil {
			fmt.Printf("Error writing workspaces.gws: %v\n", err)
		}
	}

	return wsInfo
}

func buildName(pfx, parentPath, suffix string) string {
	if parentPath == "" {
		if suffix == "" {
			return pfx
		}
		return fmt.Sprintf("%s-%s", pfx, suffix)
	}
	if suffix == "" {
		return fmt.Sprintf("%s-%s", pfx, parentPath)
	}
	return fmt.Sprintf("%s-%s-%s", pfx, parentPath, suffix)
}

func writeProjectsFile(gwsDir string, projects []projectInfo) error {
	filePath := filepath.Join(gwsDir, "projects."+gws.FileExtension)
	var lines []string

	for _, p := range projects {
		line := fmt.Sprintf("%s | %s origin", p.Name, p.RemoteURL)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(filePath, []byte(content), 0644)
}

func writeWorkspacesFile(gwsDir string, workspaces []workspaceInfo) error {
	if len(workspaces) == 0 {
		return nil
	}

	filePath := filepath.Join(gwsDir, "workspaces."+gws.FileExtension)
	var lines []string

	for _, w := range workspaces {
		line := fmt.Sprintf("%s | %s origin", w.Name, w.RemoteURL)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(filePath, []byte(content), 0644)
}

func createProjectRepo(dir string, proj projectInfo) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	readme := generateProjectReadme(proj)
	readmePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readme), 0644); err != nil {
		return err
	}

	if err := runGitInit(dir); err != nil {
		return err
	}

	if err := runGitRemoteAdd(dir, "origin", proj.RemoteURL); err != nil {
		return err
	}

	if err := runGitAddCommit(dir, "Initial commit"); err != nil {
		return err
	}

	return nil
}

func createWorkspaceRepo(dir string, ws workspaceInfo) error {
	readme := generateWorkspaceReadme(ws)
	readmePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readme), 0644); err != nil {
		return err
	}

	if err := runGitInit(dir); err != nil {
		return err
	}

	if err := runGitRemoteAdd(dir, "origin", ws.RemoteURL); err != nil {
		return err
	}

	if err := runGitAddCommit(dir, "Initial commit"); err != nil {
		return err
	}

	return nil
}

func createRootReadme(dir string, ws workspaceInfo) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", prefix))
	sb.WriteString("Generated by `gogws dev generate`\n\n")
	sb.WriteString("## Info\n\n")
	sb.WriteString("- **Type:** Root Workspace\n")
	sb.WriteString(fmt.Sprintf("- **Base URL:** %s\n", baseURL))
	sb.WriteString(fmt.Sprintf("- **Max Depth:** %d\n", maxDepth))
	sb.WriteString("\n")

	if len(ws.Projects) > 0 {
		sb.WriteString(fmt.Sprintf("## Projects (%d)\n\n", len(ws.Projects)))
		for _, p := range ws.Projects {
			sb.WriteString(fmt.Sprintf("- %s\n", p.Name))
		}
		sb.WriteString("\n")
	}

	if len(ws.Workspaces) > 0 {
		sb.WriteString(fmt.Sprintf("## Workspaces (%d)\n\n", len(ws.Workspaces)))
		for _, w := range ws.Workspaces {
			sb.WriteString(fmt.Sprintf("- %s\n", w.Name))
		}
		sb.WriteString("\n")
	}

	readmePath := filepath.Join(dir, "README.md")
	return os.WriteFile(readmePath, []byte(sb.String()), 0644)
}

func generateProjectReadme(proj projectInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", proj.Name))
	sb.WriteString("Generated by `gogws dev generate`\n\n")
	sb.WriteString("## Info\n\n")
	sb.WriteString("- **Type:** Project\n")
	sb.WriteString(fmt.Sprintf("- **Depth:** %d\n", proj.Depth))

	if proj.ParentPath == "" {
		sb.WriteString("- **Parent workspace:** (root)\n")
	} else {
		sb.WriteString(fmt.Sprintf("- **Parent workspace:** %s-%s\n", prefix, proj.ParentPath))
	}

	sb.WriteString(fmt.Sprintf("- **Full path:** %s\n", proj.Name))
	sb.WriteString(fmt.Sprintf("- **Remote:** %s\n", proj.RemoteURL))

	return sb.String()
}

func generateWorkspaceReadme(ws workspaceInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", ws.Name))
	sb.WriteString("Generated by `gogws dev generate`\n\n")
	sb.WriteString("## Info\n\n")
	sb.WriteString("- **Type:** Workspace\n")
	sb.WriteString(fmt.Sprintf("- **Depth:** %d\n", ws.Depth))

	if ws.ParentPath == "" || !strings.Contains(ws.ParentPath, "-") {
		parentName := "(root)"
		if ws.Depth > 0 {
			parts := strings.Split(ws.ParentPath, "-")
			if len(parts) > 1 {
				parentName = fmt.Sprintf("%s-%s", prefix, strings.Join(parts[:len(parts)-1], "-"))
			}
		}
		sb.WriteString(fmt.Sprintf("- **Parent workspace:** %s\n", parentName))
	} else {
		parts := strings.Split(ws.ParentPath, "-")
		if len(parts) > 1 {
			parentName := fmt.Sprintf("%s-%s", prefix, strings.Join(parts[:len(parts)-1], "-"))
			sb.WriteString(fmt.Sprintf("- **Parent workspace:** %s\n", parentName))
		} else {
			sb.WriteString("- **Parent workspace:** (root)\n")
		}
	}

	sb.WriteString(fmt.Sprintf("- **Full path:** %s\n", ws.Name))
	sb.WriteString(fmt.Sprintf("- **Remote:** %s\n", ws.RemoteURL))
	sb.WriteString("\n")

	if len(ws.Projects) > 0 {
		sb.WriteString(fmt.Sprintf("## Projects (%d)\n\n", len(ws.Projects)))
		for _, p := range ws.Projects {
			sb.WriteString(fmt.Sprintf("- %s\n", p.Name))
		}
		sb.WriteString("\n")
	}

	if len(ws.Workspaces) > 0 {
		sb.WriteString(fmt.Sprintf("## Workspaces (%d)\n\n", len(ws.Workspaces)))
		for _, w := range ws.Workspaces {
			sb.WriteString(fmt.Sprintf("- %s\n", w.Name))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func runGitInit(dir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func runGitRemoteAdd(dir, name, url string) error {
	cmd := exec.Command("git", "remote", "add", name, url)
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func runGitAddCommit(dir, message string) error {
	addCmd := exec.Command("git", "add", "-A")
	addCmd.Dir = dir
	if err := addCmd.Run(); err != nil {
		return err
	}

	commitCmd := exec.Command("git", "commit", "-m", message)
	commitCmd.Dir = dir
	commitCmd.Stdout = nil
	commitCmd.Stderr = nil
	return commitCmd.Run()
}

func printStats(ws workspaceInfo) {
	totalProjects := countProjects(ws)
	totalWorkspaces := countWorkspaces(ws)

	fmt.Printf("\nStatistics:\n")
	fmt.Printf("  Total projects: %d\n", totalProjects)
	fmt.Printf("  Total workspaces: %d\n", totalWorkspaces)
}

func countProjects(ws workspaceInfo) int {
	count := len(ws.Projects)
	for _, child := range ws.Workspaces {
		count += countProjects(child)
	}
	return count
}

func countWorkspaces(ws workspaceInfo) int {
	count := len(ws.Workspaces)
	for _, child := range ws.Workspaces {
		count += countWorkspaces(child)
	}
	return count
}
