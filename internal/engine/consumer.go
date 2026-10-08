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
	for e := range events {
		switch e.Type {
		case EventJobStart:
			slog.Info("JOB STARTED", "goroutine", e.GoroutineID, "jobId", e.JobNameId)
		case EventJobPhase:
			if e.Log != "" {
				slog.Info("JOB PHASE", "goroutine", e.GoroutineID, "jobId", e.JobNameId, "phase", e.Log)
			}
		case EventJobProgress:
			if e.Progress != nil && milestones.reached(e.JobNameId, e.Progress) {
				slog.Info("JOB PROGRESS", "jobId", e.JobNameId, "phase", e.Progress.Phase, "percent", e.Progress.Percent, "current", e.Progress.Current, "total", e.Progress.Total)
			}
		case EventJobLog:
			slog.Debug("JOB LOG", "goroutine", e.GoroutineID, "jobId", e.JobNameId, "log", e.Log)
		case EventJobErr:
			slog.Error("JOB ERR", "goroutine", e.GoroutineID, "jobId", e.JobNameId, "log", e.Log)
		case EventJobEnd:
			delete(milestones, e.JobNameId)
			if e.Success {
				slog.Info("JOB END", "goroutine", e.GoroutineID, "jobId", e.JobNameId)
			} else {
				slog.Error("JOB FAIL", "goroutine", e.GoroutineID, "jobId", e.JobNameId, "error", e.Err)
			}
		}
	}
}
