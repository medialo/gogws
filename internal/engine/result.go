package engine

import (
	"sort"
	"time"
)

type Result struct {
	Label      string
	Success    bool
	Error      error
	Duration   time.Duration
	Skipped    bool
	SkipReason string
	order      int
}

func (r *Result) IsSuccess() bool {
	return r.Success && !r.Skipped
}

func (r *Result) IsFailure() bool {
	return !r.Success && !r.Skipped
}

func (r *Result) IsSkipped() bool {
	return r.Skipped
}

type ExecuteResult struct {
	Results       []Result
	TotalDuration time.Duration
	Stopped       bool
	StopReason    string
}

func NewExecuteResult() *ExecuteResult {
	return &ExecuteResult{
		Results: make([]Result, 0),
	}
}

func (r *ExecuteResult) AddResult(result Result) {
	r.Results = append(r.Results, result)
}

func (r *ExecuteResult) SortByOrder() {
	sort.Slice(r.Results, func(i, j int) bool {
		return r.Results[i].order < r.Results[j].order
	})
}

func (r *ExecuteResult) Succeeded() []Result {
	var results []Result
	for _, res := range r.Results {
		if res.IsSuccess() {
			results = append(results, res)
		}
	}
	return results
}

func (r *ExecuteResult) Failed() []Result {
	var results []Result
	for _, res := range r.Results {
		if res.IsFailure() {
			results = append(results, res)
		}
	}
	return results
}

func (r *ExecuteResult) Skipped() []Result {
	var results []Result
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
	var labels []string
	for _, res := range r.Succeeded() {
		labels = append(labels, res.Label)
	}
	return labels
}

func (r *ExecuteResult) FailedLabels() []string {
	var labels []string
	for _, res := range r.Failed() {
		labels = append(labels, res.Label)
	}
	return labels
}

func (r *ExecuteResult) SkippedLabels() []string {
	var labels []string
	for _, res := range r.Skipped() {
		labels = append(labels, res.Label)
	}
	return labels
}
