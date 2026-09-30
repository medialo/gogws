package utils

import "fmt"

type ShellType int

//go:generate enumer -type=ShellType
const (
	Bash ShellType = iota
	Zsh
	Fish
	Powershell
)

var SnippetScriptInit = map[ShellType]func(str string) string{
	Bash:       posixScriptInit,
	Zsh:        posixScriptInit,
	Fish:       fishScriptInit,
	Powershell: powershellScriptInit,
}

func posixScriptInit(command string) string {
	return fmt.Sprintf("eval -- \"$(%s)\"", command)
}

func fishScriptInit(command string) string {
	return fmt.Sprintf("source (%s | psub)", command)
}

func powershellScriptInit(command string) string {
	return fmt.Sprintf("Invoke-Expression (& %s | Out-String)", command)
}
