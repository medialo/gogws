package engine

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Notify represent a func that can be call inside a job to send event
type Notify func(eventType EventType, log string)

type JobFunction func(ctx context.Context, notify Notify) error

type Job struct {
	JobNameId string
	Fn        JobFunction
}

func NewJob(label string, fn JobFunction) Job {
	return Job{JobNameId: label, Fn: fn}
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

func (nc *NotifiableCmd) Run(todo context.Context, notify Notify) error {
	var buf bytes.Buffer
	stdout := &lineNotifyWriter{notify: func(s string) { notify(EventJobLog, s) }}
	stderr := &lineNotifyWriter{notify: func(s string) {
		buf.WriteString(s)
		buf.WriteString("\n")
		notify(EventJobLog, s)
	}}
	nc.cmd.Stdout = stdout
	nc.cmd.Stderr = stderr

	if err := nc.cmd.Start(); err != nil {
		return err
	}

	err := nc.cmd.Wait()

	stdout.Close()
	stderr.Close()

	if err == nil {
		return nil
	}
	return fmt.Errorf("\uE0B0\uE0B0 %s %s \n> %w", nc.cmd.String(), &buf, err)
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
