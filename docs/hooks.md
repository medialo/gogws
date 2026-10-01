# Hooks

Scripts that gogws runs before or after a command.

## Where

```
<workspace>/.gws/hooks/      local hooks, need your approval
~/.gws/hooks/                global hooks, always trusted
```

Inside either folder, a `windows/`, `linux/` or `darwin/` subfolder holds hooks for that OS only. They replace the hook of the same name in the parent folder:

```
.gws/hooks/
├── post-ff.sh             every OS without a dedicated hook
├── api.post-ff.sh
└── windows/
    ├── post-ff.ps1        used on Windows instead of post-ff.sh
    └── api.post-ff.ps1
```

For a given hook, the first folder that has it is used: `<workspace>/.gws/hooks/<os>/`, `<workspace>/.gws/hooks/`, `~/.gws/hooks/<os>/`, `~/.gws/hooks/`.

## Names

`[<project>.]<hook>[.<ext>]`

| Hook | When |
|---|---|
| `pre-init`, `post-init` | `init projects`, `init provider` |
| `pre-update`, `post-update` | `update` |
| `pre-clone`, `post-clone` | each repository cloned by `clone` or `update` |
| `pre-fetch`, `post-fetch` | `fetch` |
| `pre-ff`, `post-ff` | `ff` |
| `pre-check`, `post-check` | `check` |

| Extension | Runs with |
|---|---|
| none | executed directly; on Windows, the shebang line picks the interpreter |
| `.sh`, `.bash` | `bash` (Git for Windows' bash on Windows) |
| `.ps1` | `pwsh`, or `powershell` |
| `.py` | `python3`, or `python` |
| `.cmd`, `.bat` | `cmd /c` |
| `.js` | `node` |

Without a project prefix, a hook runs once for the command, in the workspace folder, and its output goes straight to the terminal. `pre-clone` and `post-clone` are the exception: they run once per cloned repository.

## Project hooks

`api.post-ff.sh` runs only for the project named `api` in that workspace's `.projects.gws`. It does not apply to projects of sub-workspaces; they have their own `.gws/hooks/`.

It runs inside the project's job, in the project folder:

- `pre-*` before the git command, `post-*` after it, only if it succeeded;
- the worker line shows `⚙ hook post-ff` and the hook's last output line;
- a non-zero exit code fails that project.

Project hooks are local only.

## Exit codes

A failing workspace `pre-*` hook stops the command before it starts. A failing `post-*` hook makes the command exit with an error. A failing project hook fails its project.

## Conflicts

Only one file per hook and project in a folder. `api.post-ff` next to `api.post-ff.sh`, or `windows/post-ff.ps1` next to `windows/post-ff.cmd`, is a conflict. The command stops and lists the files:

```
conflicting hooks for api.post-ff in .gws/hooks/windows: api.post-ff.cmd, api.post-ff.ps1 — keep only one, then check with 'gogws doctor run HookConflict'
```

`gogws doctor run` reports conflicts for every OS folder of every workspace.

## Trust

Global hooks always run. A local or project hook runs once you approve that exact file: gogws stores its path and SHA-256 in `~/.gws/trusted-hooks.yaml`. A new or modified file asks again:

```
[hook:project] Hook 'post-ff' found at: /work/ws/.gws/hooks/api.post-ff.sh
Workspace: /work/ws
This hook file is modified since it was trusted.

Options:
  [r] Run this hook once
  [s] Skip this hook
  [t] Run and trust this file (asked again if it changes)
Choose [r/s/t]:
```

Questions are asked before the command starts. `--trust-hooks=all` runs untrusted hooks without asking, `--trust-hooks=skip` skips them.

## Environment

| Variable | |
|---|---|
| `GOGWS_COMMAND` | `ff`, `fetch`, `update`... |
| `GOGWS_WORKSPACE_DIR` | workspace root |
| `GOGWS_HOOK_NAME` | `post-ff`... |
| `GOGWS_HOOK_ORIGIN` | `global`, `local` or `project` |
| `GOGWS_PROJECT_DIR`, `GOGWS_PROJECT_NAME` | project hooks and clone hooks |
| `GOGWS_PROJECTS` | `post-init`, `post-update`, `post-check`: one path per line |
| `GOGWS_FETCHED`, `GOGWS_PULLED` | `post-fetch`, `post-ff`: number of repositories that succeeded |

## Examples

`.gws/hooks/post-update.sh`:

```bash
#!/usr/bin/env bash
printf '%s\n' "$GOGWS_PROJECTS" | while read -r dir; do
  [ -n "$dir" ] && [ -f "$dir/package.json" ] && (cd "$dir" && npm ci)
done
```

`.gws/hooks/api.post-ff.sh`:

```bash
#!/usr/bin/env bash
make generate
```
