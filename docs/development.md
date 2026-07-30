# Development Guide

Guide for contributing to gogws development.

## Prerequisites

- **Go 1.21+**
- **Git**
- **Make** (optional, for convenience commands)

## Getting Started

```bash
# Clone the repository
git clone https://github.com/medialo/gogws.git
cd gogws

# Build
go build -o gogws ./cmd/gogws

# Run tests
go test ./...

# Run the binary
./gogws --help
```

## Building

### Development Build

```bash
go build -o gogws ./cmd/gogws
```

### With Make

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Clean build artifacts
make clean
```

### Cross-Compilation

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o gogws-linux-amd64 ./cmd/gogws

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o gogws-darwin-amd64 ./cmd/gogws

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o gogws-darwin-arm64 ./cmd/gogws

# Windows
GOOS=windows GOARCH=amd64 go build -o gogws-windows-amd64.exe ./cmd/gogws
```

## Project Structure

```
gogws/
├── cmd/
│   └── gogws/
│       └── main.go              # Entry point
│
├── internal/
│   ├── commands/                # CLI commands (Cobra)
│   │   ├── commands.go          # Command registration
│   │   ├── root/
│   │   │   └── root.go          # Root command + global flags
│   │   ├── initcmd/
│   │   │   ├── init.go          # Parent init command
│   │   │   ├── projects.go      # init projects
│   │   │   ├── workspaces.go    # init workspaces
│   │   │   └── gitignore.go     # init gitignore
│   │   ├── status/
│   │   │   └── status.go
│   │   ├── fetch/
│   │   │   └── fetch.go
│   │   ├── ff/
│   │   │   └── ff.go
│   │   ├── update/
│   │   │   └── update.go
│   │   ├── clone/
│   │   │   └── clone.go
│   │   ├── check/
│   │   │   └── check.go
│   │   ├── configcmd/
│   │   │   └── config.go
│   │   └── version/
│   │       └── version.go
│   │
│   ├── engine/                  # Parallel execution engine
│   │   ├── command.go           # RepoCommand struct + factories
│   │   ├── engine.go            # Execute (parallel/serial)
│   │   ├── result.go            # Result types
│   │   ├── shell.go             # Git/shell command execution
│   │   └── output.go            # Output handler
│   │
│   ├── gws/                     # Workspace management
│   │   ├── types.go             # Core types (Project, Workspace)
│   │   ├── parser.go            # File parsing
│   │   └── loader.go            # Workspace loading
│   │
│   ├── git/                     # Git operations
│   │   ├── clone.go
│   │   ├── status.go
│   │   └── remote.go
│   │
│   ├── hooks/                   # Hook system
│   │   ├── executor.go          # Hook execution
│   │   └── trust.go             # Trust management
│   │
│   ├── config/                  # Configuration
│   │   ├── config.go            # Main config
│   │   └── user.go              # User config (~/.gws/)
│   │
│   ├── gitignore/               # Gitignore generation
│   │   ├── gitignore.go
│   │   ├── template.go
│   │   └── default.tmpl.go
│   │
│   └── ui/                      # User interface
│       ├── cli/
│       │   └── renderer.go      # CLI output rendering
│       └── styles/
│           └── styles.go        # Lipgloss styles
│
├── docs/                        # Documentation
│
└── Makefile
```

## Key Components

### Commands (Cobra)

Each command is in its own package under `internal/commands/`:

```go
// internal/commands/mycommand/mycommand.go
package mycommand

import (
    "github.com/medialo/gogws/internal/config"
    "github.com/spf13/cobra"
)

func NewCommand(getConfig func() *config.Config) *cobra.Command {
    return &cobra.Command{
        Use:   "mycommand",
        Short: "Short description",
        Long:  `Long description`,
        RunE: func(cmd *cobra.Command, args []string) error {
            return runMyCommand(getConfig)
        },
    }
}

func runMyCommand(getConfig func() *config.Config) error {
    cfg := getConfig()
    if cfg == nil {
        return fmt.Errorf("no workspace found")
    }
    
    // Implementation
    return nil
}
```

Register in `internal/commands/commands.go`:

```go
rootCmd.AddCommand(mycommand.NewCommand(root.GetConfig))
```

### Execution Engine

The engine handles parallel/serial execution across repositories:

```go
import "github.com/medialo/gogws/internal/engine"

// Create commands
commands := []engine.RepoCommand{
    engine.NewGitCommand(repoPath, repoName, "fetch", "--all"),
    engine.NewGitCommand(repoPath2, repoName2, "fetch", "--all"),
}

// Execute (parallel or serial based on options)
result := engine.Execute(commands, engine.ExecuteOptions{
    Parallel:    cfg.Parallel,
    StopOnError: cfg.StopOnError,
})

// Render results
output := engine.NewOutputHandler(renderer, verbose)
output.RenderSummary(result, "Fetched")
```

**Command types:**
- `NewGitCommand(path, name, args...)` — Run git command
- `NewShellCommand(path, name, command)` — Run shell command
- `NewCustomCommand(path, name, fn)` — Run custom function

### Workspace Loading

```go
import "github.com/medialo/gogws/internal/gws"

// Load workspace
ws, err := gws.New(workspaceRoot).Load()

// Access projects
for _, project := range ws.Projects {
    fmt.Println(project.Path, project.Remotes)
}

// Access nested workspaces
for _, child := range ws.Children {
    fmt.Println(child.Path, child.Projects)
}

// Find missing
missing := ws.MissingProjects()
```

### Hooks

```go
import "github.com/medialo/gogws/internal/hooks"

// Run pre-command hook
if err := hooks.PreFetch(workspaceRoot); err != nil {
    return fmt.Errorf("pre-fetch hook failed: %w", err)
}

// Run post-command hook
if err := hooks.PostFetch(workspaceRoot, successCount); err != nil {
    return fmt.Errorf("post-fetch hook failed: %w", err)
}
```

## Testing

### Run All Tests

```bash
go test ./...
```

### Run Specific Package

```bash
go test ./internal/gws/...
```

### With Coverage

```bash
go test -cover ./...

# HTML report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Verbose

```bash
go test -v ./...
```

## Adding a New Command

1. **Create package:**
   ```bash
   mkdir -p internal/commands/mycommand
   ```

2. **Implement command:**
   ```go
   // internal/commands/mycommand/mycommand.go
   package mycommand
   
   import (
       "fmt"
       "github.com/medialo/gogws/internal/config"
       "github.com/medialo/gogws/internal/engine"
       "github.com/medialo/gogws/internal/gws"
       "github.com/medialo/gogws/internal/hooks"
       "github.com/medialo/gogws/internal/ui/cli"
       "github.com/spf13/cobra"
   )
   
   func NewCommand(getConfig func() *config.Config) *cobra.Command {
       return &cobra.Command{
           Use:   "mycommand",
           Short: "Do something",
           RunE: func(cmd *cobra.Command, args []string) error {
               return run(getConfig)
           },
       }
   }
   
   func run(getConfig func() *config.Config) error {
       cfg := getConfig()
       if cfg == nil {
           return fmt.Errorf("no workspace found")
       }
       
       // Pre-hook
       if err := hooks.PreMyCommand(cfg.WorkspaceRoot); err != nil {
           return err
       }
       
       // Load workspace
       ws, err := gws.New(cfg.WorkspaceRoot).Load()
       if err != nil {
           return err
       }
       
       // Build commands
       var commands []engine.RepoCommand
       for _, p := range ws.Projects {
           commands = append(commands, engine.NewGitCommand(
               filepath.Join(cfg.WorkspaceRoot, p.Path),
               p.Path,
               "my-git-command",
           ))
       }
       
       // Execute
       result := engine.Execute(commands, engine.ExecuteOptions{
           Parallel:    cfg.Parallel,
           StopOnError: cfg.StopOnError,
       })
       
       // Render
       renderer := cli.NewRenderer()
       output := engine.NewOutputHandler(renderer, false)
       output.RenderSummary(result, "Completed")
       
       // Post-hook
       return hooks.PostMyCommand(cfg.WorkspaceRoot, result.SuccessCount())
   }
   ```

3. **Register command:**
   ```go
   // internal/commands/commands.go
   import "github.com/medialo/gogws/internal/commands/mycommand"
   
   // In RegisterCommands():
   rootCmd.AddCommand(mycommand.NewCommand(root.GetConfig))
   ```

4. **Add hooks (optional):**
   ```go
   // internal/hooks/executor.go
   func PreMyCommand(workspaceRoot string) error {
       return executeHook(workspaceRoot, "pre-mycommand", "mycommand", nil)
   }
   
   func PostMyCommand(workspaceRoot string, count int) error {
       return executeHook(workspaceRoot, "post-mycommand", "mycommand", map[string]string{
           "GOGWS_COUNT": strconv.Itoa(count),
       })
   }
   ```

## Code Style

- **No comments** unless absolutely necessary
- **Use existing patterns** from similar commands
- **Run formatters** before committing:
  ```bash
  go fmt ./...
  go vet ./...
  ```

## Dependencies

Key dependencies:
- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [Viper](https://github.com/spf13/viper) — Configuration
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — Terminal styling
- [go-git](https://github.com/go-git/go-git) — Git operations (library mode)

## Releasing

1. Update `CHANGELOG.md`
2. Update version in code (if hardcoded)
3. Tag release:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```
4. Build release binaries:
   ```bash
   make build-all
   ```

## Debugging

### Verbose Mode

```bash
./gogws fetch --verbose --parallel=1
```

### With Delve

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug
dlv debug ./cmd/gogws -- status
```

### Environment

Useful environment variables for debugging:

```bash
export GOGWS_DEBUG=1
export GOGWS_PARALLEL=1
```
