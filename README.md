# gogws

Manage multiple repositories as one workspace with one cli: status, fetch, pull and clone them all in parallel.

⭐⭐⭐ Don't forget to give a star if you like the project, it help a lot  ⭐⭐⭐

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

## Quick install

```bash
go install github.com/medialo/gogws/cmd/gogws@latest
gogws --version
```

## Full documentation

You can find the documention for gogws [here](LICENSE) 

## Feature request or a bug ?

If you need a particular feature, or you have found a bug, please create an issue 

## License

[AGPL-3.0](LICENSE)
