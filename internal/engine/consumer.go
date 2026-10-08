package engine

import (
	"log/slog"
)

const progressMilestone = 25

type progressMilestones map[string]struct {
	phase     string
	milestone int
}

func (m progressMilestones) reached(jobID string, p *Progress) bool {
	last, seen := m[jobID]
	milestone := p.Percent / progressMilestone * progressMilestone
	if seen && last.phase == p.Phase && last.milestone == milestone {
		return false
	}
	m[jobID] = struct {
		phase     string
		milestone int
	}{phase: p.Phase, milestone: milestone}
	return true
}

func ConsumeVerbose(events <-chan Event) {
	milestones := progressMilestones{}
	for event := range events {
		switch e := event.(type) {
		case JobStarted:
			slog.Info("JOB STARTED", "goroutine", e.Worker, "jobId", e.Job)
		case JobPhase:
			if e.Phase != "" {
				slog.Info("JOB PHASE", "goroutine", e.Worker, "jobId", e.Job, "phase", e.Phase)
			}
		case JobProgress:
			if milestones.reached(e.Job, &e.Progress) {
				slog.Info("JOB PROGRESS", "jobId", e.Job, "phase", e.Phase, "percent", e.Percent, "current", e.Current, "total", e.Total)
			}
		case JobLog:
			slog.Debug("JOB LOG", "goroutine", e.Worker, "jobId", e.Job, "log", e.Line)
		case JobEnded:
			delete(milestones, e.Job)
			if e.Success {
				slog.Info("JOB END", "goroutine", e.Worker, "jobId", e.Job)
			} else {
				slog.Error("JOB FAIL", "goroutine", e.Worker, "jobId", e.Job, "error", e.Err)
			}
		}
	}
}
