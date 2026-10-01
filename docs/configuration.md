# Configuration

gogws uses a layered configuration system with multiple sources.

## Configuration Priority

Configuration values are resolved in this order (highest to lowest priority):

1. **Command-line flags** — `--parallel=10`
2. **Environment variables** — `GOGWS_PARALLEL=10`
3. **User config file** — `~/.gws/config.yaml`
4. **Defaults**

## File Locations

### User Configuration

```
~/.gws/
├── config.yaml              # User settings
├── hooks/                   # Global hooks
│   ├── pre-fetch
│   └── post-update
└── templates/
    └── gitignore.tmpl       # Custom gitignore template
```

### Workspace Configuration

gogws supports two locations for workspace configuration. The `.gws/` directory is preferred:

```
my-workspace/
├── .gws/                    # ✅ Preferred location
│   ├── projects.gws         # Repository definitions
│   ├── workspaces.gws       # Nested workspaces
│   └── hooks/               # Local hooks
├── .projects.gws            # ⚠️ Legacy (still supported)
├── .workspaces.gws          # ⚠️ Legacy
└── .ignore.gws              # Ignore patterns
```

If both `.gws/projects.gws` and `.projects.gws` exist, the `.gws/` version takes priority and a warning is displayed.

---

## Projects File

The projects file defines repositories in your workspace.

### Location

- **Preferred:** `.gws/projects.gws`
- **Legacy:** `.projects.gws`

### Syntax

```
# Comments start with #
path/to/repo | remote-url [remote-name]
path/to/repo | url1 [name1] | url2 [name2]
```

- **path** — Relative path from workspace root
- **remote-url** — Git clone URL (SSH or HTTPS)
- **remote-name** — Optional, defaults to `origin`

### Examples

```bash
# Simple project
api | git@github.com:company/api.git

# With explicit remote name
frontend | git@github.com:company/frontend.git origin

# Multiple remotes (fork workflow)
my-fork | git@github.com:me/project.git origin | git@github.com:upstream/project.git upstream

# Nested path
libs/shared | git@github.com:company/shared-lib.git

# HTTPS URL
public-repo | https://github.com/user/repo.git
```

---

## Workspaces File

The workspaces file defines nested workspaces.

### Location

- **Preferred:** `.gws/workspaces.gws`
- **Legacy:** `.workspaces.gws`

### Syntax

```
path | remote-url [remote-name]
```

### Example

```bash
# Nested workspaces
team-a | git@github.com:company/team-a-workspace.git
team-b | git@github.com:company/team-b-workspace.git
shared | git@github.com:company/shared-workspace.git origin
```

See [Nested Workspaces](workspaces.md) for more details.

---

## Ignore File

The ignore file contains regex patterns for repositories to ignore.

### Location

`.ignore.gws` (at workspace root)

### Syntax

Each line is a Go regular expression pattern:

```
# Ignore vendor directories
^vendor/

# Ignore anything ending with -test
-test$

# Ignore specific paths
^experiments/
^temp/
```

### Usage

Ignored repositories are:
- Excluded from `gogws status` output
- Skipped by `gogws check` (shown in "Ignored" category)
- Not cloned by `gogws update`

---

## User Configuration

User-level settings stored in `~/.gws/config.yaml`.

### Options

```yaml
# Trusted workspace paths (for local hooks)
trusted-workspaces:
  - "/home/user/work/*"
  - "/home/user/personal/**"
  - "/opt/company/repos"
```

### trusted-workspaces

> **Deprecated:** this list no longer grants trust to hooks. Local and project hooks are now trusted per file (path + SHA-256) in `~/.gws/trusted-hooks.yaml`, see [Hooks](hooks.md#per-file-trust).

List of workspace paths where local hooks were automatically trusted.

**Wildcard patterns:**
- `*` — Matches one directory level
- `**` — Matches any number of directory levels

**Examples:**

```yaml
trusted-workspaces:
  # Trust all direct children of /home/user/work
  - "/home/user/work/*"
  
  # Trust all descendants of /home/user/personal
  - "/home/user/personal/**"
  
  # Trust a specific workspace
  - "/opt/company/main-workspace"
```

### Managing Configuration

```bash
# List available keys
gogws config list

# Get a value
gogws config get trusted-workspaces

# Add a trusted workspace
gogws config set trusted-workspaces /home/user/work/*
```

---

## Environment Variables

### General

| Variable | Description | Example |
|----------|-------------|---------|
| `GOGWS_PARALLEL` | Default parallel workers | `10` |
| `GOGWS_FORMAT` | Default output format | `json` |
| `NO_COLOR` | Disable colored output | `1` |

### Example

```bash
# CI/CD environment
export GOGWS_PARALLEL=20
export GOGWS_FORMAT=json
export NO_COLOR=1

gogws status > status.json
```

---

## Custom Gitignore Template

Create a custom template for generated `.gitignore` files.

### Location

`~/.gws/templates/gitignore.tmpl`

### Template Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `{{.ConfigDir}}` | Config directory name | `.gws` |
| `{{.ProjectsFile}}` | Projects file name | `.projects.gws` |
| `{{.WorkspacesFile}}` | Workspaces file name | `.workspaces.gws` |
| `{{.Extension}}` | File extension | `gws` |

### Default Template

```
# GWS Workspace - ignore all sub-projects and workspaces
*

# But track GWS configuration files
!.gitignore
!README.md

# Track config directory
!{{.ConfigDir}}/
!{{.ConfigDir}}/**

# Track legacy files at root
!{{.ProjectsFile}}
!{{.WorkspacesFile}}
!*.{{.Extension}}
```

### Custom Template Example

```
# My custom GWS workspace
*

# Track configuration
!.gitignore
!README.md
!{{.ConfigDir}}/
!{{.ConfigDir}}/**

# Also track documentation
!docs/
!docs/**

# Track CI configuration
!.github/
!.github/**
```

---

## Themes

Customize colors and styles with a theme file.

### Location

Specified via `--theme` flag or in user config.

### Format

```yaml
colors:
  title: "63"       # ANSI color code
  success: "46"
  warning: "214"
  error: "196"
  info: "39"
  subtle: "241"
  path: "33"
  branch: "141"
  remote: "220"
  status: "46"
  stats: "250"

styles:
  title_bold: true
  path_bold: true
```

### Usage

```bash
gogws status --theme=~/.config/gogws/mytheme.yaml
```

---

## Complete Example

### User Config (`~/.gws/config.yaml`)

```yaml
trusted-workspaces:
  - "/home/dev/work/**"
  - "/home/dev/personal/*"
```

### Workspace Config (`.gws/projects.gws`)

```bash
# Backend services
api          | git@github.com:company/api.git
auth-service | git@github.com:company/auth.git
worker       | git@github.com:company/worker.git

# Frontend
web          | git@github.com:company/web.git
mobile       | git@github.com:company/mobile.git

# Shared libraries
libs/common  | git@github.com:company/common.git origin | git@github.com:upstream/common.git upstream
libs/utils   | git@github.com:company/utils.git
```

### Nested Workspaces (`.gws/workspaces.gws`)

```bash
team-platform | git@github.com:company/platform-workspace.git
team-mobile   | git@github.com:company/mobile-workspace.git
```

### Ignore File (`.ignore.gws`)

```
^vendor/
^node_modules/
-backup$
^temp/
```
