package engine

import (
	"log/slog"
)

func ConsumeVerbose(events <-chan Event) {
	for e := range events {
		switch e.Type {
		case EventJobStart:
			slog.Info("JOB STARTED", "goroutine", e.GoroutineID, "jobId", e.JobNameId)
		case EventJobLog:
			slog.Info("JOB LOG", "goroutine", e.GoroutineID, "jobId", e.JobNameId, "log", e.Log)
		case EventJobErr:
			slog.Error("JOB ERR", "goroutine", e.GoroutineID, "jobId", e.JobNameId, "log", e.Log)
		case EventJobEnd:
			if e.Success {
				slog.Info("JOB END", "goroutine", e.GoroutineID, "jobId", e.JobNameId)
			} else {
				slog.Error("JOB FAIL", "goroutine", e.GoroutineID, "jobId", e.JobNameId, "error", e.Err)
			}
		}
	}
}
