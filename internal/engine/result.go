package engine

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type JobResult struct {
	JobId      string
	Success    bool
	Error      error
	Duration   time.Duration
	Skipped    bool
	SkipReason string
	order      int
}

func (r *JobResult) IsSuccess() bool {
	return r.Success && !r.Skipped
}

func (r *JobResult) IsFailure() bool {
	return !r.Success && !r.Skipped
}

func (r *JobResult) IsSkipped() bool {
	return r.Skipped
}

type ExecutionResult struct {
	Results        []JobResult
	aTotalDuration atomic.Int64
	Stopped        bool
	StopReason     string
	mu             sync.Mutex
	successCount   int
	failedCount    int
	skippedCount   int
}

func NewNoExecutionResult() *ExecutionResult {
	return &ExecutionResult{
		Results:        make([]JobResult, 0),
		Stopped:        true,
		aTotalDuration: atomic.Int64{},
		StopReason:     "No jobs to execute",
	}
}

func NewExecuteResult(nb int) *ExecutionResult {
	return &ExecutionResult{
		Results:        make([]JobResult, 0, nb),
		aTotalDuration: atomic.Int64{},
		Stopped:        false,
		StopReason:     "",
	}
}

func (r *ExecutionResult) AddResult(result JobResult) {
	r.mu.Lock()
	r.Results = append(r.Results, result)
	switch {
	case result.IsSkipped():
		r.skippedCount++
	case result.IsSuccess():
		r.successCount++
	default:
		r.failedCount++
	}
	r.mu.Unlock()
	r.aTotalDuration.Add(int64(result.Duration))
}

func (r *ExecutionResult) TotalDuration() time.Duration {
	return time.Duration(r.aTotalDuration.Load())
}

func (r *ExecutionResult) SortByOrder() {
	sort.Slice(r.Results, func(i, j int) bool {
		return r.Results[i].order < r.Results[j].order
	})
}

func (r *ExecutionResult) filter(count int, keep func(*JobResult) bool) []JobResult {
	if count == 0 {
		return nil
	}
	results := make([]JobResult, 0, count)
	for i := range r.Results {
		if keep(&r.Results[i]) {
			results = append(results, r.Results[i])
		}
	}
	return results
}

func (r *ExecutionResult) labels(count int, keep func(*JobResult) bool) []string {
	labels := make([]string, 0, count)
	for i := range r.Results {
		if !keep(&r.Results[i]) {
			continue
		}
		if label, err := labelString(r.Results[i].JobId); err == nil {
			labels = append(labels, label)
		}
	}
	return labels
}

func (r *ExecutionResult) Succeeded() []JobResult {
	return r.filter(r.SuccessCount(), (*JobResult).IsSuccess)
}

func (r *ExecutionResult) Failed() []JobResult {
	return r.filter(r.FailedCount(), (*JobResult).IsFailure)
}

func (r *ExecutionResult) Skipped() []JobResult {
	return r.filter(r.SkippedCount(), (*JobResult).IsSkipped)
}

func (r *ExecutionResult) SuccessCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.successCount
}

func (r *ExecutionResult) FailedCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.failedCount
}

func (r *ExecutionResult) SkippedCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.skippedCount
}

func (r *ExecutionResult) TotalCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.Results)
}

func (r *ExecutionResult) HasErrors() bool {
	return r.FailedCount() > 0
}

func (r *ExecutionResult) AllSucceeded() bool {
	return r.FailedCount() == 0 && r.SuccessCount() > 0
}

func (r *ExecutionResult) SuccessLabels() []string {
	return r.labels(r.SuccessCount(), (*JobResult).IsSuccess)
}

func (r *ExecutionResult) FailedLabels() []string {
	return r.labels(r.FailedCount(), (*JobResult).IsFailure)
}

func (r *ExecutionResult) SkippedLabels() []string {
	return r.labels(r.SkippedCount(), (*JobResult).IsSkipped)
}

// todo a delete
func labelString(label any) (string, error) {
	switch v := label.(type) {
	case string:
		return v, nil
	case fmt.Stringer:
		return v.String(), nil
	default:
		return "", fmt.Errorf("unsupported label type %T", label)
	}
}
