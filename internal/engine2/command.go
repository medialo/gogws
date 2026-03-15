package engine2

import (
	"context"
	"os/exec"
)

//type CommandType int

//const (
//	CommandTypeGit CommandType = iota
//	CommandTypeShell
//	CommandTypeFunc
//)

//type BaseCommand struct {
//	Path     string // where the command is executed
//	RepoName string
//	Type     CommandType
//	Args     []string
//	Context  map[string]any
//	order    int
//}

//type FuncCommand struct {
//	BaseCommand
//	Action func() (any, error)
//}

type EngineCommandNotify = func(eventType EngineCommandEventType, log string)

type EngineCommand func(ctx context.Context, notify EngineCommandNotify)

type EngineCommandEventType int

const (
	LOG EngineCommandEventType = iota
	OK
	ERR
)

type EngineCommandEvent struct {
	id        int
	log       string
	eventType EngineCommandEventType
	err       error
}

type NotifiableCmd struct {
	cmd *exec.Cmd
}

// Wrap wrap any *exec.Cmd to *NotifiableCmd
func Wrap(cmd *exec.Cmd) *NotifiableCmd {
	return &NotifiableCmd{cmd: cmd}
}

func (nc *NotifiableCmd) Run(ctx context.Context, notify EngineCommandNotify) error {
	stdout := &lineNotifyWriter{notify: func(s string) { notify(LOG, s) }}
	stderr := &lineNotifyWriter{notify: func(s string) { notify(ERR, s) }}
	nc.cmd.Stdout = stdout
	nc.cmd.Stderr = stderr
	err := nc.cmd.Start()
	if err != nil {
		return err
	}
	err = nc.cmd.Wait()
	if err != nil {
		return err
	}
	stdout.Close() // vide les éventuels résidus de buffer
	stderr.Close()
	return err
}

type lineNotifyWriter struct {
	notify func(string)
	buf    []byte
}

func (lnw *lineNotifyWriter) Write(p []byte) (int, error) {
	lnw.buf = append(lnw.buf, p...)
	var last int
	for i, b := range lnw.buf {
		if b == '\n' {
			lnw.notify(string(lnw.buf[last:i]))
			last = i + 1
		}
	}
	lnw.buf = lnw.buf[last:]
	return len(p), nil
}

func (lnw *lineNotifyWriter) Close() error {
	if len(lnw.buf) > 0 {
		lnw.notify(string(lnw.buf))
		lnw.buf = nil
	}
	return nil
}

//func NewGitCommand(repoPath, repoName string, args ...string) BaseCommand {
//	return BaseCommand{
//		Path:     repoPath,
//		RepoName: repoName,
//		Type:     CommandTypeGit,
//		Args:     args,
//		Context:  make(map[string]any),
//	}
//}
//
//func NewShellCommand(repoPath, repoName string, args ...string) BaseCommand {
//	return BaseCommand{
//		Path:     repoPath,
//		RepoName: repoName,
//		Type:     CommandTypeShell,
//		Args:     args,
//		Context:  make(map[string]any),
//	}
//}
//
//func NewFuncCommand(repoPath, repoName string, action func() (any, error)) FuncCommand {
//	return FuncCommand{
//		BaseCommand: BaseCommand{
//			Path:     repoPath,
//			RepoName: repoName,
//			Type:     CommandTypeFunc,
//			Args:     []string{},
//			Context:  make(map[string]any),
//		},
//		Action: action,
//	}
//}
