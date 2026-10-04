# Commands

Run from anywhere inside a workspace; gogws walks up to the first folder that has a `.projects.gws` or `.workspaces.gws`. Use `-D <dir>` to point at another workspace.

## Global flags

| Flag | |
|---|---|
| `-D, --working-dir <dir>` | workspace to use instead of the current directory |
| `--parallel <n>` | number of repositories handled at once (default 5) |
| `--stop-on-error` | stop at the first failing repository |
| `-c, --only-changes` | `status`: hide clean repositories |
| `--format text\|json\|yaml` | `status` output format |
| `--trust-hooks ask\|all\|skip` | what to do with untrusted hooks, see [hooks](hooks.md#trust) |
| `--no-color` | plain output (`NO_COLOR` works too) |
| `-v` | debug logs, repeat for more (`-vv`, `-vvv`) |

## Create a workspace

```bash
gogws init                          # same as: gogws init projects
gogws init projects                 # scan sub-folders for git repositories, write .gws/.projects.gws
gogws init workspaces               # pick which sub-folders are workspaces, write .gws/.workspaces.gws
gogws init provider github:my-org   # build the files from a GitHub org or GitLab group (beta)
gogws init gitignore                # .gitignore that ignores everything except the .gws files
```

`init gitignore` is for a workspace that is itself a git repository: only its `.gws` files get committed, not the cloned projects.

`--reset` overwrites existing files. `init gitignore` also takes `--force` and `--remove`.

Add entries one by one:

```bash
gogws add project git@github.com:org/api.git api
gogws add workspace git@github.com:org/tools.git tools
gogws add workspace --folder scratch              # plain folder, no remote
```

Missing arguments are prompted. `--auto-clone` clones right away.

From inside an already cloned repository, register it in the enclosing workspace:

```bash
gogws add current                                  # prompts Project (default) or Workspace
gogws add current --workspace                      # no prompt, add as workspace
gogws add current --project                        # no prompt, add as project
```

Remotes are read from the repository. A repository already listed in the projects or workspaces file is rejected.

## Daily use

```bash
gogws status        # or: gogws st, or just: gogws
gogws fetch
gogws ff            # git pull --ff-only in every repository
```

`fetch` and `ff` run in parallel and show one line per worker. Project hooks show up there too:

```
─── Workers ───
  ⠋ [0] .../api  ⚙ hook post-ff (api.post-ff.sh)
        running api checks...
  ⠋ [1] .../web  Fast-forwarding...
```

`status --format json` (or `yaml`) prints the same data for scripts.

## Clone what is missing

```bash
gogws update                 # clone missing projects and workspaces of this workspace
gogws update -r              # also inside the workspaces it clones, until nothing is missing
gogws update --prune         # drop entries whose repository no longer exists upstream
gogws clone api web          # clone only these projects (paths as written in .projects.gws)
```

Provider workspaces, beta (`github:`, `gitlab:`, `gitlab-graphql:`, see [workspace files](workspace.md#github-and-gitlab)):

| Flag | |
|---|---|
| `--refresh-providers` | re-read orgs/groups older than `provider-cache-ttl` |
| `--force-refresh-providers` | re-read all of them now |
| `--no-provider-discovery` | treat them as plain git remotes |

`--skip-projects` / `--skip-workspaces` limit what `update` clones.

## Find and jump

```bash
$ gogws search i
   Kind    Name       Path
 ✓ project api        ~/work/my-workspace/api
 ✓ project cli        ~/work/my-workspace/tools/cli
 ✓ project infra      ~/work/my-workspace/tools/infra
 ✓ project shared-lib ~/work/my-workspace/shared-lib
4 match(es)
```

`--full-path` matches the whole path instead of the name. `--cd` prints the path when there is exactly one match (and asks when there are several), which is what the `gcd` helper uses:

```bash
# bash / zsh, in ~/.bashrc or ~/.zshrc
eval "$(gogws search --completion bash)"
# fish
gogws search --completion fish | source
# PowerShell, in $PROFILE
gogws search --completion powershell | Invoke-Expression

gcd infra        # cd into tools/infra
```

`--alias j,go` names the helper something else.

## Check a workspace

```bash
gogws check                 # repositories on disk vs. listed in the .gws files
gogws check --show-known    # also list the known ones
gogws doctor run            # run every check, in every nested workspace
gogws doctor run HookConflict
gogws doctor run --autofix  # fix what can be fixed (duplicates)
gogws doctor list
```

| Check | Fix |
|---|---|
| `DuplicateWorkspace` | auto |
| `DuplicateProject` | auto |
| `HookConflict` | manual, see [hooks](hooks.md#conflicts) |

## Other

```bash
gogws config                            # show ~/.gws/config.yaml
gogws config list                       # available keys
gogws config get provider-cache-ttl
gogws config set provider-cache-ttl 12h
gogws completion bash|zsh|fish|powershell
gogws version
```
