package gitignore

const DefaultTemplate = `# GWS Workspace - ignore all sub-projects and workspaces
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

hash.gws
{{.ConfigDir}}/hash.gws
`
