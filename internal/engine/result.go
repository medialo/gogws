package engine

import (
	"fmt"
	"sort"
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

type ExecuteResult struct {
	Results        []JobResult
	aTotalDuration atomic.Int64
	Stopped        bool
	StopReason     string
}

func NewNoExecutionResult() *ExecuteResult {
	return &ExecuteResult{
		Results:        make([]JobResult, 0),
		Stopped:        true,
		aTotalDuration: atomic.Int64{},
		StopReason:     "No jobs to execute",
	}
}

func NewExecuteResult(nb int) *ExecuteResult {
	return &ExecuteResult{
		Results:        make([]JobResult, 0, nb),
		aTotalDuration: atomic.Int64{},
		Stopped:        false,
		StopReason:     "",
	}
}

func (r *ExecuteResult) AddResult(result JobResult) {
	r.Results = append(r.Results, result)
	r.aTotalDuration.Add(int64(result.Duration))
}

func (r *ExecuteResult) TotalDuration() time.Duration {
	return time.Duration(r.aTotalDuration.Load())
}

func (r *ExecuteResult) SortByOrder() {
	sort.Slice(r.Results, func(i, j int) bool {
		return r.Results[i].order < r.Results[j].order
	})
}

func (r *ExecuteResult) Succeeded() []JobResult {
	var results []JobResult
	for _, res := range r.Results {
		if res.IsSuccess() {
			results = append(results, res)
		}
	}
	return results
}

func (r *ExecuteResult) Failed() []JobResult {
	var results []JobResult
	for _, res := range r.Results {
		if res.IsFailure() {
			results = append(results, res)
		}
	}
	return results
}

func (r *ExecuteResult) Skipped() []JobResult {
	var results []JobResult
	for _, res := range r.Results {
		if res.IsSkipped() {
			results = append(results, res)
		}
	}
	return results
}

func (r *ExecuteResult) SuccessCount() int {
	return len(r.Succeeded())
}

func (r *ExecuteResult) FailedCount() int {
	return len(r.Failed())
}

func (r *ExecuteResult) SkippedCount() int {
	return len(r.Skipped())
}

func (r *ExecuteResult) TotalCount() int {
	return len(r.Results)
}

func (r *ExecuteResult) HasErrors() bool {
	return r.FailedCount() > 0
}

func (r *ExecuteResult) AllSucceeded() bool {
	return r.FailedCount() == 0 && r.SuccessCount() > 0
}

func (r *ExecuteResult) SuccessLabels() []string {
	labels := make([]string, 0, len(r.Succeeded()))
	for _, res := range r.Succeeded() {
		if label, err := labelString(res.JobId); err == nil {
			labels = append(labels, label)
		}
	}
	return labels
}

func (r *ExecuteResult) FailedLabels() []string {
	labels := make([]string, 0, len(r.Failed()))
	for _, res := range r.Failed() {
		if label, err := labelString(res.JobId); err == nil {
			labels = append(labels, label)
		}
	}
	return labels
}

func (r *ExecuteResult) SkippedLabels() []string {
	labels := make([]string, 0, len(r.Skipped()))
	for _, res := range r.Skipped() {
		if label, err := labelString(res.JobId); err == nil {
			labels = append(labels, label)
		}
	}
	return labels
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
