package engine

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/medialo/gogws/internal/git"
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

func (nc *NotifiableCmd) Run(ctx context.Context, notify Notify) error {
	var buf bytes.Buffer
	progress := func(line string) { notify(EventJobProgress, line) }
	stdout := &lineNotifyWriter{notify: func(s string) { notify(EventJobLog, s) }, progress: progress}
	stderr := &lineNotifyWriter{notify: func(s string) {
		buf.WriteString(s)
		buf.WriteString("\n")
		notify(EventJobLog, s)
	}, progress: progress}
	nc.cmd.Stdout = stdout
	nc.cmd.Stderr = stderr
	git.WithUnattendedEnv(nc.cmd)

	if err := nc.cmd.Start(); err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = nc.cmd.Process.Kill()
		case <-done:
		}
	}()

	err := nc.cmd.Wait()
	close(done)

	stdout.Close()
	stderr.Close()

	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return fmt.Errorf("%s cancelled: %w", nc.cmd.String(), ctx.Err())
	}
	if hint := git.AuthenticationHint(buf.String()); hint != "" {
		return fmt.Errorf("\uE0B0\uE0B0 %s %s \n> %w (%s)", nc.cmd.String(), &buf, err, hint)
	}
	return fmt.Errorf("\uE0B0\uE0B0 %s %s \n> %w", nc.cmd.String(), &buf, err)
}

const progressInterval = 100 * time.Millisecond

type lineNotifyWriter struct {
	notify       func(string)
	progress     func(string)
	buf          []byte
	now          func() time.Time
	lastProgress time.Time
	pending      string
	lastPhase    string
	lastPercent  int
}

func (lnw *lineNotifyWriter) Write(p []byte) (int, error) {
	lnw.buf = append(lnw.buf, p...)

	for {
		idx := bytes.IndexAny(lnw.buf, "\r\n")
		if idx == -1 {
			break
		}

		lnw.line(string(lnw.buf[:idx]), lnw.buf[idx] == '\r')
		lnw.buf = lnw.buf[idx+1:]
	}

	return len(p), nil
}

func (lnw *lineNotifyWriter) line(line string, carriageReturn bool) {
	if len(line) == 0 {
		return
	}
	if lnw.progress != nil {
		if p, ok := parseProgress(line); ok {
			lnw.pending = ""
			if p.Phase != lnw.lastPhase || p.Percent != lnw.lastPercent {
				lnw.lastPhase = p.Phase
				lnw.lastPercent = p.Percent
				lnw.progress(line)
			}
			return
		}
	}
	if carriageReturn {
		lnw.throttled(line)
		return
	}
	lnw.pending = ""
	lnw.notify(line)
}

func (lnw *lineNotifyWriter) throttled(line string) {
	now := lnw.clock()
	if now.Sub(lnw.lastProgress) < progressInterval {
		lnw.pending = line
		return
	}
	lnw.lastProgress = now
	lnw.pending = ""
	lnw.notify(line)
}

func (lnw *lineNotifyWriter) clock() time.Time {
	if lnw.now != nil {
		return lnw.now()
	}
	return time.Now()
}

func (lnw *lineNotifyWriter) Close() error {
	if len(lnw.buf) > 0 {
		lnw.line(string(lnw.buf), false)
		lnw.buf = nil
	}
	if lnw.pending != "" {
		lnw.notify(lnw.pending)
		lnw.pending = ""
	}
	return nil
}
