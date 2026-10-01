# Hooks

Hooks allow you to run custom scripts at various points during gogws command execution. They follow a git-like convention with executable files discovered automatically.

## Hook Locations

Hooks can be defined at two levels:

### Global Hooks (User-level)

Located in `~/.gws/hooks/`. These hooks are **always trusted** and run automatically.

```
~/.gws/
└── hooks/
    ├── pre-init
    ├── post-init
    ├── pre-update
    └── ...
```

### Local Hooks (Workspace-level)

Located in `<workspace>/.gws/hooks/`. These hooks require trust verification before execution.

```
<workspace>/
└── .gws/
    └── hooks/
        ├── pre-init
        ├── post-init
        └── ...
```

### Project Hooks

A hook can target a single project by prefixing its name with the project name, as written in the `.projects.gws` file of the workspace that declares it:

```
<workspace>/
├── .projects.gws        # api | git@github.com:org/api.git
└── .gws/
    └── hooks/
        ├── post-ff          # workspace hook, runs once after `gogws ff`
        └── api.post-ff.sh   # project hook, runs inside the `api` job
```

- Project hooks are **not recursive**: a hook in `<workspace>/.gws/hooks/` only applies to projects declared directly by that workspace, never to projects of nested workspaces (they use their own `.gws/hooks/`).
- They run **inside the project job**, on the same worker, right after the project operation succeeds (`post-*`) or right before it (`pre-*`). The worker shows `⚙ hook post-ff` with the last line printed by the hook below it.
- The exit code decides the job result: a non-zero exit marks this project as failed in the summary.
- Supported for `fetch`, `ff` and `clone`/`update` (`pre-clone`, `post-clone`). For clone, the workspace `pre-clone`/`post-clone` hooks also run inside each project job.
- Project hooks are local only (no `~/.gws/hooks/<name>.<hook>`).

## OS-Specific Hooks

A hook can be specialized for an OS by placing it in a `windows`, `linux` or `darwin` subfolder of a `hooks/` directory (local or global). Files directly in `hooks/` are the catch-all for every OS without a dedicated hook:

```
.gws/hooks/
├── api.post-ff.sh        # catch-all: linux, darwin, ...
├── post-ff.sh            # workspace catch-all
└── windows/
    ├── api.post-ff.ps1   # used instead of api.post-ff.sh on Windows
    └── post-ff.ps1
```

File names are the same in every folder: `[<project>.]<hook>[.<ext>]`.

## Hook Priority

For a given hook (and project), gogws picks the first folder that contains it:

1. `<workspace>/.gws/hooks/<os>/`
2. `<workspace>/.gws/hooks/`
3. `~/.gws/hooks/<os>/` (workspace hooks only)
4. `~/.gws/hooks/` (workspace hooks only)

Local hooks therefore **override** global hooks, and an OS folder overrides the catch-all of the same level.

## Conflicts

Only **one file per hook, project and folder** is allowed. `api.post-ff` + `api.post-ff.sh`, or `windows/api.post-ff.ps1` + `windows/api.post-ff.cmd`, are conflicts; `windows/api.post-ff.ps1` + `api.post-ff.sh` is not.

A command that needs a conflicting hook stops before running anything and lists the files:

```
conflicting hooks for api.post-ff in .gws/hooks/windows: api.post-ff.cmd, api.post-ff.ps1 — keep only one, then check with 'gogws doctor run HookConflict'
```

`gogws doctor run` reports every conflict of every workspace, for all OS folders (not only the current OS):

```
 ✗ .   HookConflict   [FAIL]
         windows/api.post-ff: api.post-ff.cmd, api.post-ff.ps1
```

Conflicts in `~/.gws/hooks` are only reported when a command needs the hook.

## File Names and Interpreters

A hook file is either named exactly like the hook (`post-ff`) or uses a known extension (`post-ff.sh`). Both forms count as the same hook, so having both is a conflict.

| Extension | Runs with |
|-----------|-----------|
| `.sh` | `bash` (fallback `sh`; on Windows, Git for Windows bash is used, never the WSL launcher) |
| `.bash` | `bash` |
| `.ps1` | `pwsh -NoProfile -File` (fallback `powershell`) |
| `.py` | `python3` (fallback `python`) |
| `.cmd`, `.bat` | `cmd /c` |
| `.js` | `node` |
| none | executed directly on Linux/macOS; on Windows the shebang (`#!/usr/bin/env bash`) selects the interpreter |

Other suffixes (`.sample`, `.bak`, ...) are ignored.

## Available Hooks

| Hook | Trigger | Description |
|------|---------|-------------|
| `pre-init` | Before init command | Before workspace initialization |
| `post-init` | After init command | After workspace initialization |
| `pre-update` | Before update command | Before cloning missing repos |
| `post-update` | After update command | After cloning missing repos |
| `pre-clone` | Before cloning a repository | Before each individual clone |
| `post-clone` | After cloning a repository | After each individual clone |
| `pre-fetch` | Before fetch command | Before fetching all repos |
| `post-fetch` | After fetch command | After fetching all repos |
| `pre-ff` | Before fast-forward pull | Before pulling all repos |
| `post-ff` | After fast-forward pull | After pulling all repos |
| `pre-check` | Before check command | Before workspace check |
| `post-check` | After check command | After workspace check |

## Trust System

Local hooks from external/cloned workspaces are not automatically trusted for security reasons.

### Trust Mode Flag

Use the `--trust-hooks` flag to control behavior:

| Mode | Description |
|------|-------------|
| `ask` (default) | Prompt for each untrusted hook with options to run, skip, or trust |
| `all` | Run all hooks without prompting |
| `skip` | Skip all untrusted local hooks |

Example:
```bash
gogws update --trust-hooks=skip
gogws fetch --trust-hooks=all
```

### Per-File Trust

Local and project hooks are trusted **per file**: gogws records the absolute path and the SHA-256 of each approved hook in `~/.gws/trusted-hooks.yaml`.

```yaml
hooks:
  - path: /home/user/work/ws/.gws/hooks/api.post-ff.sh
    sha256: f5efb3416d6b7844609f37518abe6dcedb180dbae78b8627e56c428ba6e0d28b
```

- A **new** hook file is never trusted automatically, even in a workspace you trusted before.
- A **modified** hook file (e.g. changed by a `git pull`) must be approved again.
- `trusted-workspaces` in `~/.gws/config.yaml` is deprecated and no longer grants trust.

### Interactive Trust Prompt

When `--trust-hooks=ask` (default) and an untrusted hook is found, gogws asks before starting the command (never while the progress UI is running):

```
[hook:project] Hook 'post-ff' found at: /path/to/workspace/.gws/hooks/api.post-ff.sh
Workspace: /path/to/workspace
This hook file is modified since it was trusted.

Options:
  [r] Run this hook once
  [s] Skip this hook
  [t] Run and trust this file (asked again if it changes)
Choose [r/s/t]:
```

## Environment Variables

All hooks receive these environment variables:

| Variable | Description |
|----------|-------------|
| `GOGWS_COMMAND` | The command being executed (`init`, `update`, `clone`, `fetch`, `ff`, `check`) |
| `GOGWS_WORKSPACE_DIR` | The absolute path to the workspace root directory |
| `GOGWS_WORKSPACE` | Deprecated alias of `GOGWS_WORKSPACE_DIR` |
| `GOGWS_HOOK_NAME` | The name of the hook being executed |
| `GOGWS_HOOK_ORIGIN` | The origin of the hook (`global`, `local` or `project`) |
| `GOGWS_PROJECT_DIR` | Project hooks and clone hooks: absolute path of the project directory |
| `GOGWS_PROJECT_NAME` | Project hooks and clone hooks: project name from the `.gws` file |
| `GOGWS_PROJECTS` | Hooks with a project list (`post-init`, `post-update`, `post-check`): one path per line |
| `GOGWS_FETCHED`, `GOGWS_PULLED` | `post-fetch` / `post-ff`: number of successful repositories |

## Execution Behavior

- Hooks run **synchronously** - the command waits for the hook to complete
- Workspace hooks run in the **workspace root directory**; project hooks run in the **project directory** (the owning workspace directory if the project is not cloned yet)
- Hooks inherit the **current environment** plus gogws-specific variables
- Workspace hooks print a start line (`[hook:local] post-ff <path>`), their output, then `✓ post-ff 1.2s` or `✗ post-ff exit 1`
- Project hook output is streamed into the job: the worker shows the running hook and its last output line
- If a hook **exits with non-zero**: a `pre-*` workspace hook aborts the command, a `post-*` workspace hook makes the command fail, a project hook fails its project job
- If a hook file **doesn't exist**, it is silently skipped (run with `-v` to see the searched paths)

## Creating Hooks

1. Create the hooks directory:
```bash
# Global hooks
mkdir -p ~/.gws/hooks

# Local hooks (workspace-level)
mkdir -p .gws/hooks
```

2. Create your hook script (git-style name, or with an extension such as `.sh`/`.ps1`):
```bash
#!/bin/bash

echo "Running $GOGWS_COMMAND in $GOGWS_WORKSPACE_DIR"
echo "Hook: $GOGWS_HOOK_NAME (origin: $GOGWS_HOOK_ORIGIN)"

# Example: Send notification after update
if [ "$GOGWS_COMMAND" = "update" ]; then
    echo "Workspace updated!"
fi
```

3. Make it executable:
```bash
chmod +x ~/.gws/hooks/post-update
# or
chmod +x .gws/hooks/post-update
```

## Configuration Files Location

Configuration files can now be placed in the `.gws/` directory:

```
<workspace>/
├── .gws/
│   ├── hooks/           # Hook scripts
│   ├── projects.gws     # Projects definition (preferred)
│   └── workspaces.gws   # Workspaces definition (preferred)
├── .projects.gws        # Legacy location (warning if both exist)
└── .workspaces.gws      # Legacy location (warning if both exist)
```

If both `.gws/projects.gws` and `.projects.gws` exist, the `.gws/` version takes priority and a warning is displayed asking to remove the legacy file.

## Examples

### Post-Update: Slack Notification

Send a Slack notification when repositories are cloned:

```bash
#!/bin/bash
# ~/.gws/hooks/post-update

if [ -n "$SLACK_WEBHOOK" ]; then
  curl -s -X POST "$SLACK_WEBHOOK" \
    -H 'Content-type: application/json' \
    -d "{
      \"text\": \"Workspace updated\",
      \"blocks\": [
        {
          \"type\": \"section\",
          \"text\": {
            \"type\": \"mrkdwn\",
            \"text\": \"*Workspace updated*\n$GOGWS_WORKSPACE\"
          }
        }
      ]
    }"
fi
```

### Pre-Fetch: VPN Check

Ensure VPN is connected before fetching from private repositories:

```bash
#!/bin/bash
# .gws/hooks/pre-fetch

INTERNAL_HOST="git.internal.company.com"

if ! ping -c1 -W2 "$INTERNAL_HOST" &>/dev/null; then
  echo "ERROR: Cannot reach $INTERNAL_HOST"
  echo "Please connect to VPN before fetching."
  exit 1
fi

echo "VPN connection verified"
```

### Post-Clone: Auto-Install Dependencies

Automatically install dependencies for cloned repositories:

```bash
#!/bin/bash
# .gws/hooks/post-clone

echo "Installing dependencies for cloned repositories..."

for dir in */; do
  [ -d "$dir/.git" ] || continue
  
  if [ -f "$dir/package.json" ]; then
    echo "→ npm install in $dir"
    (cd "$dir" && npm ci --silent)
  elif [ -f "$dir/go.mod" ]; then
    echo "→ go mod download in $dir"
    (cd "$dir" && go mod download)
  elif [ -f "$dir/requirements.txt" ]; then
    echo "→ pip install in $dir"
    (cd "$dir" && pip install -q -r requirements.txt)
  elif [ -f "$dir/Gemfile" ]; then
    echo "→ bundle install in $dir"
    (cd "$dir" && bundle install --quiet)
  fi
done

echo "Dependencies installed"
```

### Post-Update: Run Setup Scripts

Run project-specific setup scripts after cloning:

```bash
#!/bin/bash
# .gws/hooks/post-update

for dir in */; do
  [ -d "$dir/.git" ] || continue
  
  if [ -x "$dir/scripts/setup.sh" ]; then
    echo "Running setup for $dir"
    (cd "$dir" && ./scripts/setup.sh)
  fi
done
```

### Pre-FF: Check for Uncommitted Changes

Warn before pulling if there are uncommitted changes:

```bash
#!/bin/bash
# ~/.gws/hooks/pre-ff

DIRTY_REPOS=""

for dir in */; do
  [ -d "$dir/.git" ] || continue
  
  if ! git -C "$dir" diff --quiet 2>/dev/null; then
    DIRTY_REPOS="$DIRTY_REPOS  - $dir\n"
  fi
done

if [ -n "$DIRTY_REPOS" ]; then
  echo "WARNING: The following repositories have uncommitted changes:"
  echo -e "$DIRTY_REPOS"
  echo "These will NOT be pulled to avoid conflicts."
fi
```

## Troubleshooting

### Hook Not Executing

1. **Check file is executable:**
   ```bash
   ls -la .gws/hooks/
   chmod +x .gws/hooks/your-hook
   ```

2. **Check hook name matches exactly:**
   Hook names must match exactly, optionally with a supported extension:
   - ✅ `pre-fetch`
   - ✅ `pre-fetch.sh`
   - ✅ `api.pre-fetch.ps1` (project hook)
   - ✅ `windows/pre-fetch.ps1` (Windows only)
   - ❌ `pre_fetch`
   - ❌ `pre-fetch.sample`
   - ❌ `win/pre-fetch.ps1` (OS folders are `windows`, `linux`, `darwin`)

### Conflicting Hooks

```
conflicting hooks for api.post-ff in .gws/hooks: api.post-ff.js, api.post-ff.sh — keep only one, then check with 'gogws doctor run HookConflict'
```

Keep a single file per hook, project and folder (move an OS-specific variant into `windows/`, `linux/` or `darwin/`), then run `gogws doctor run HookConflict` until it passes.

3. **Run with verbose mode:**
   ```bash
   gogws fetch --verbose
   ```
   This shows hook discovery and execution.

4. **Check shebang line:**
   Ensure the first line is a valid interpreter:
   ```bash
   #!/bin/bash
   # or
   #!/usr/bin/env bash
   ```

### Permission Denied

```
Error: permission denied: .gws/hooks/pre-fetch
```

**Solution:**
```bash
chmod +x .gws/hooks/pre-fetch
```

On Windows, use an extension (`.sh`, `.ps1`, `.cmd`, ...) or a shebang so gogws can pick the interpreter.

### Trust Issues

**"This hook file is new, never trusted" / "modified since it was trusted"**

Options:
1. **One-time run:** Choose `[r]` at the prompt
2. **Trust this file:** Choose `[t]`; its SHA-256 is stored in `~/.gws/trusted-hooks.yaml` and you are asked again only if it changes
3. **Skip hooks:** Use `--trust-hooks=skip`

**Bypass for CI/CD:**
```bash
gogws update --trust-hooks=all
```

### Hook Fails with Exit Code

If a hook exits with non-zero, the parent command fails:

```
[hook:local] Running pre-fetch...
ERROR: VPN not connected
Error: pre-fetch hook failed: exit status 1
```

To continue despite hook failures, modify your hook to return 0:

```bash
#!/bin/bash
# Non-blocking hook
do_something || echo "Warning: do_something failed"
exit 0  # Always succeed
```

### Debugging Hooks

Add debug output to your hook:

```bash
#!/bin/bash
set -x  # Print commands as they execute

echo "GOGWS_COMMAND: $GOGWS_COMMAND"
echo "GOGWS_WORKSPACE: $GOGWS_WORKSPACE"
echo "GOGWS_HOOK_NAME: $GOGWS_HOOK_NAME"
echo "GOGWS_HOOK_ORIGIN: $GOGWS_HOOK_ORIGIN"
echo "PWD: $(pwd)"

# Your hook logic here
```
