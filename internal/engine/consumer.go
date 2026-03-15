package engine

import (
	"log/slog"
)

func ConsumeVerbose(events <-chan Event) {
	for e := range events {
		switch e.Type {
		case EventJobStart:
			slog.Info("job started", "goroutine", e.GoroutineID, "job", e.JobLabel)
		case EventLog:
			slog.Debug("job log", "goroutine", e.GoroutineID, "job", e.JobLabel, "log", e.Log)
		case EventErr:
			slog.Warn("job error output", "goroutine", e.GoroutineID, "job", e.JobLabel, "log", e.Log)
		case EventJobEnd:
			if e.Success {
				slog.Info("job completed", "goroutine", e.GoroutineID, "job", e.JobLabel)
			} else {
				slog.Error("job failed", "goroutine", e.GoroutineID, "job", e.JobLabel, "error", e.Err)
			}
		}
	}
}
