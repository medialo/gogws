package engineui

import (
	tea "charm.land/bubbletea/v2"
)

type LogEvent string

type UILog struct {
	LogsCh chan LogEvent
}

func NewUILog() *UILog {
	return &UILog{
		LogsCh: make(chan LogEvent, 100),
	}
}

func (l *UILog) Write(p []byte) (n int, err error) {
	msg := string(p[:])

	l.LogsCh <- LogEvent(msg)

	return len(p), nil
}

func (l *UILog) WaitFor() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-l.LogsCh
		if !ok {
			return struct{}{}
		}
		return msg
	}
}

func (l *UILog) Close() error {
	close(l.LogsCh)
	l.LogsCh = nil
	return nil
}
