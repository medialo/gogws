# gogws

Manage a folder full of Git repositories as one workspace: status, fetch, pull and clone them all in parallel.

gogws reads the same `.projects.gws` files as [gws](https://github.com/StreakyCobra/gws).

```
$ gogws status
╭───────────────────────────────────────────╮
│  GOGWS - Workspace Status - my-workspace  │
╰───────────────────────────────────────────╯

  Workspace
  📁 my-workspace [folder]
  ╰── 📁 tools [folder]

  Projects
  ● api  1 uncommitted, 1 untracked
    * main                 origin/main               =
  ● web
    * main                 origin/main               ↓1
  ○ shared-lib (not cloned)

┌──────────────────────────────────────────────────────────┐
│   Projects: 3  │  Clean: 0  │  Changed: 2  │  Missing: 1 │
│ Workspaces: 1  │  Clean: 2  │  Changed: 0  │  Missing: 0 │
└──────────────────────────────────────────────────────────┘
```

## Install

```bash
go install github.com/medialo/gogws/cmd/gogws@latest
```

Or from a clone: `make build` (binary in `bin/`), `make test`.

## Quick start

```bash
cd ~/work
gogws init        # find the git repositories below and write .gws/.projects.gws
gogws status      # state of every repository
gogws fetch       # git fetch everywhere
gogws ff          # fast-forward pull everywhere
gogws update      # clone what is listed but missing
```

Commit `.gws/` so the next person runs `gogws update` and gets the same tree.

## Docs

- [Commands](docs/usage.md)
- [Workspace files](docs/workspace.md): `.projects.gws`, nested workspaces, GitHub/GitLab discovery (beta), user config
- [Hooks](docs/hooks.md): run scripts before/after commands, per project, per OS

## License

[AGPL-3.0](LICENSE)
