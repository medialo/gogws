package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

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
