package engine

import (
	"bytes"
	"context"
	"os/exec"
)

type Notify func(eventType EventType, log string)

type CommandFunc func(ctx context.Context, notify Notify) error

type Job struct {
	Label string
	Fn    CommandFunc
}

func NewJob(label string, fn CommandFunc) Job {
	return Job{Label: label, Fn: fn}
}

func WrapRunner(notify Notify) func(context.Context, *exec.Cmd) error {
	return func(ctx context.Context, cmd *exec.Cmd) error {
		return Wrap(cmd).Run(ctx, notify)
	}
}

type NotifiableCmd struct {
	cmd *exec.Cmd
}

func Wrap(cmd *exec.Cmd) *NotifiableCmd {
	return &NotifiableCmd{cmd: cmd}
}

func (nc *NotifiableCmd) Run(ctx context.Context, notify Notify) error {
	stdout := &lineNotifyWriter{notify: func(s string) { notify(EventLog, s) }}
	stderr := &lineNotifyWriter{notify: func(s string) { notify(EventLog, s) }}
	nc.cmd.Stdout = stdout
	nc.cmd.Stderr = stderr

	if err := nc.cmd.Start(); err != nil {
		return err
	}

	err := nc.cmd.Wait()

	stdout.Close()
	stderr.Close()

	return err
}

type lineNotifyWriter struct {
	notify func(string)
	buf    []byte
}

func (lnw *lineNotifyWriter) Write(p []byte) (int, error) {
	lnw.buf = append(lnw.buf, p...)

	for {
		idxN := bytes.IndexByte(lnw.buf, '\n')
		idxR := bytes.IndexByte(lnw.buf, '\r')

		var idx int
		if idxN == -1 && idxR == -1 {
			break
		} else if idxN == -1 {
			idx = idxR
		} else if idxR == -1 {
			idx = idxN
		} else {
			idx = min(idxN, idxR)
		}

		line := string(lnw.buf[:idx])
		if len(line) > 0 {
			lnw.notify(line)
		}
		lnw.buf = lnw.buf[idx+1:]
	}

	return len(p), nil
}

func (lnw *lineNotifyWriter) Close() error {
	if len(lnw.buf) > 0 {
		lnw.notify(string(lnw.buf))
		lnw.buf = nil
	}
	return nil
}
