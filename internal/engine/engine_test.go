package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRunJobs_EmitsTypedEventSequence(t *testing.T) {
	job := NewJob("api", func(ctx context.Context, notify Notify) error {
		notify.Log("fetching")
		notify.Phase("hook post-ff")
		notify.Progress(Progress{Phase: "Receiving objects", Percent: 50, Current: 5, Total: 10})
		return errors.New("boom")
	})

	events, resultCh := NewEngine(DefaultOptions().WithParallel(1)).RunJobs(context.Background(), []Job{job})
	var kinds []string
	for event := range events {
		if event.JobID() != "api" || event.WorkerID() != 0 {
			t.Fatalf("event %T has job %q worker %d", event, event.JobID(), event.WorkerID())
		}
		switch e := event.(type) {
		case JobStarted:
			kinds = append(kinds, "start")
		case JobLog:
			kinds = append(kinds, "log:"+e.Line)
		case JobPhase:
			kinds = append(kinds, "phase:"+e.Phase)
		case JobProgress:
			kinds = append(kinds, fmt.Sprintf("progress:%s:%d", e.Phase, e.Percent))
		case JobEnded:
			kinds = append(kinds, fmt.Sprintf("end:%v:%v", e.Success, e.Err))
		}
	}
	<-resultCh

	want := "start|log:fetching|phase:hook post-ff|progress:Receiving objects:50|end:false:boom"
	if strings.Join(kinds, "|") != want {
		t.Fatalf("events = %v, want %s", kinds, want)
	}
}

func TestNotify_ZeroValueIsNoop(t *testing.T) {
	var notify Notify
	notify.Log("ignored")
	notify.Phase("ignored")
	notify.Progress(Progress{})
}

func TestRunJobs_CollectsEveryResultConcurrently(t *testing.T) {
	const nbJobs = 200
	jobs := make([]Job, 0, nbJobs)
	for i := range nbJobs {
		jobs = append(jobs, NewJob(fmt.Sprintf("job-%d", i), func(ctx context.Context, notify Notify) error {
			if i%3 == 0 {
				return errors.New("boom")
			}
			return nil
		}))
	}

	events, resultCh := NewEngine(DefaultOptions().WithParallel(8)).RunJobs(context.Background(), jobs)
	for range events {
	}
	result := <-resultCh

	if result.TotalCount() != nbJobs {
		t.Fatalf("results = %d, want %d", result.TotalCount(), nbJobs)
	}
	wantFailed := (nbJobs + 2) / 3
	if result.FailedCount() != wantFailed || result.SuccessCount() != nbJobs-wantFailed {
		t.Fatalf("success/failed = %d/%d, want %d/%d", result.SuccessCount(), result.FailedCount(), nbJobs-wantFailed, wantFailed)
	}
	if len(result.Failed()) != result.FailedCount() || len(result.SuccessLabels()) != result.SuccessCount() {
		t.Fatal("filtered slices disagree with counters")
	}
	for i, r := range result.Results {
		if r.JobId != fmt.Sprintf("job-%d", i) {
			t.Fatalf("result %d = %s, results not sorted by order", i, r.JobId)
		}
	}
}

func TestExecutionResult_SkippedCounter(t *testing.T) {
	result := NewExecuteResult(3)
	result.AddResult(JobResult{JobId: "a", Success: true})
	result.AddResult(JobResult{JobId: "b", Skipped: true})
	result.AddResult(JobResult{JobId: "c"})

	if result.SuccessCount() != 1 || result.SkippedCount() != 1 || result.FailedCount() != 1 {
		t.Fatalf("counts = %d/%d/%d", result.SuccessCount(), result.SkippedCount(), result.FailedCount())
	}
	if result.AllSucceeded() || !result.HasErrors() {
		t.Fatal("expected errors")
	}
	if labels := result.SkippedLabels(); len(labels) != 1 || labels[0] != "b" {
		t.Fatalf("skipped labels = %v", labels)
	}
}

func TestLineNotifyWriter_ThrottlesProgress(t *testing.T) {
	var lines []string
	now := time.Unix(0, 0)
	w := &lineNotifyWriter{
		notify: func(s string) { lines = append(lines, s) },
		now:    func() time.Time { return now },
	}

	for i := range 1000 {
		fmt.Fprintf(w, "Receiving objects: %d%%\r", i/10)
		now = now.Add(time.Millisecond)
	}
	if len(lines) > 12 {
		t.Fatalf("progress emitted %d events, want throttled", len(lines))
	}

	fmt.Fprint(w, "Receiving objects: 100%\r")
	w.Close()
	if last := lines[len(lines)-1]; last != "Receiving objects: 100%" {
		t.Fatalf("last line = %q, want final progress", last)
	}
}

func TestLineNotifyWriter_EmitsEveryFullLine(t *testing.T) {
	var lines []string
	w := &lineNotifyWriter{
		notify: func(s string) { lines = append(lines, s) },
		now:    func() time.Time { return time.Unix(0, 0) },
	}

	fmt.Fprint(w, "one\ntwo\nthree\npartial")
	w.Close()

	if strings.Join(lines, ",") != "one,two,three,partial" {
		t.Fatalf("lines = %v", lines)
	}
}
