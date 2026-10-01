# Workspace files

A workspace is a folder with a `.gws/` directory:

```
my-workspace/
├── .gws/
│   ├── .projects.gws      repositories cloned in this folder
│   ├── .workspaces.gws    sub-workspaces (optional)
│   └── hooks/             (optional, see hooks.md)
├── api/
├── web/
└── tools/                 a sub-workspace, with its own .gws/
    ├── .gws/.projects.gws
    ├── cli/
    └── infra/
```

The files can also sit at the workspace root (`./.projects.gws`), like in gws. If both exist, `.gws/` wins and a warning is printed.

## .projects.gws

One repository per line: a path relative to the workspace, then one or more remotes.

```
# path       | url                                  [name] | url [name] ...
api          | git@github.com:org/api.git
web          | git@github.com:org/web.git
libs/shared  | git@github.com:me/shared.git                 | git@github.com:org/shared.git upstream
```

The first remote is named `origin`, the second `upstream`, unless a name follows the URL. Empty lines and `#` comments are ignored.

## .workspaces.gws

Same format, for folders that are workspaces themselves:

```
.      | git@github.com:org/my-workspace.git
tools  | git@github.com:org/tools.git
notes  | folder
```

- `.` is the workspace's own remote. With it, the root shows up as a git repository in `gogws status`.
- `folder` means a plain directory with no remote; `update` just creates it.
- Each sub-workspace has its own `.gws/` and can nest further (10 levels max).

`status` and `fetch` work on the projects of the current workspace. `ff`, `update -r` and `doctor` go through the whole tree, and `search` finds anything in it.

## .ignore.gws

Regular expressions, one per line, matched against the absolute path of each project. Matching projects are left out of every command.

```
/archive/
-legacy$
```

The file lives at the workspace root (not in `.gws/`).

## GitHub and GitLab

> [!WARNING]
> **Beta.** Discovery through `github:`, `gitlab:` and `gitlab-graphql:` remotes is unstable: its behavior and the files it writes may change between versions.

A workspace remote can point to a whole organization or group instead of a repository:

```
company  | github:my-org
platform | gitlab:my-group/platform
infra    | gitlab-graphql:my-group/infra
```

On `gogws update`, gogws lists the repositories (and GitLab subgroups) through the provider API and writes them into `company/.gws/.projects.gws` and `.workspaces.gws`. Then they are cloned like any other entry.

- Self-hosted: put the full URL after the prefix, e.g. `gitlab:https://gitlab.example.com/my-group`.
- Tokens come from `GITHUB_TOKEN` and `GITLAB_TOKEN`. Without a token, only public repositories are visible.
- `gitlab-graphql` gives the same result as `gitlab`, through the GraphQL API.
- `gogws init provider github:my-org` does the same for the current folder.
- Discovered workspaces are not re-read on every update; see `--refresh-providers` in [commands](usage.md#clone-what-is-missing).

## User config

`~/.gws/config.yaml`:

```yaml
provider-cache-ttl: 24h   # how old a provider workspace can get before --refresh-providers re-reads it
```

Also in `~/.gws/`:

- `hooks/`: global hooks
- `trusted-hooks.yaml`: hooks you approved, written by gogws (see [trust](hooks.md#trust))
