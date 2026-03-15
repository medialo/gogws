package engine

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"gogws/internal/gws"
)

type ExecuteOptions struct {
	Parallel    int
	StopOnError bool
	Timeout     time.Duration
	Verbose     bool

	OnStart    func(cmd RepoCommand)
	OnComplete func(result Result)
	OnProgress func(current, total int, cmd RepoCommand)
}

func DefaultOptions() ExecuteOptions {
	return ExecuteOptions{
		Parallel:    0,
		StopOnError: false,
		Timeout:     0,
		Verbose:     false,
	}
}

func Execute(commands []RepoCommand, opts ExecuteOptions) *ExecuteResult {
	slog.Debug("Executing commands", "count", len(commands), "parallel", opts.Parallel)
	if len(commands) == 0 {
		return NewExecuteResult()
	}

	for i := range commands {
		commands[i].order = i
	}

	parallel := opts.Parallel
	if parallel == 0 {
		parallel = gws.DefaultParallel
	}

	if len(commands) == 1 {
		parallel = 1
	}

	if parallel == 1 {
		return executeSerial(commands, opts)
	}
	return executeParallel(commands, opts, parallel)
}

func executeSerial(commands []RepoCommand, opts ExecuteOptions) *ExecuteResult {
	execResult := NewExecuteResult()
	startTime := time.Now()

	for i, cmd := range commands {
		if opts.OnStart != nil {
			opts.OnStart(cmd)
		}

		if opts.OnProgress != nil {
			opts.OnProgress(i+1, len(commands), cmd)
		}

		result := executeSingleCommand(cmd, opts.Timeout)

		if opts.OnComplete != nil {
			opts.OnComplete(result)
		}

		execResult.AddResult(result)

		if opts.StopOnError && result.IsFailure() {
			execResult.Stopped = true
			execResult.StopReason = "stopped on first error"
			break
		}
	}

	execResult.TotalDuration = time.Since(startTime)
	return execResult
}

func executeParallel(commands []RepoCommand, opts ExecuteOptions, parallel int) *ExecuteResult {
	execResult := NewExecuteResult()
	startTime := time.Now()

	var wg sync.WaitGroup
	sem := make(chan struct{}, min(parallel, len(commands)))
	var mu sync.Mutex
	var stopped atomic.Bool
	var completedCount atomic.Int32

	for _, cmd := range commands {
		if stopped.Load() {
			break
		}

		wg.Add(1)
		go func(c RepoCommand) {
			defer wg.Done()

			if stopped.Load() {
				return
			}

			sem <- struct{}{}
			defer func() { <-sem }()

			if stopped.Load() {
				return
			}

			if opts.OnStart != nil {
				opts.OnStart(c)
			}

			result := executeSingleCommand(c, opts.Timeout)

			completed := int(completedCount.Add(1))
			if opts.OnProgress != nil {
				opts.OnProgress(completed, len(commands), c)
			}

			if opts.OnComplete != nil {
				opts.OnComplete(result)
			}

			mu.Lock()
			slog.Debug("Command completed", "repo", c.RepoName, "success", result.Success, "duration", result.Duration)
			execResult.AddResult(result)
			mu.Unlock()

			if opts.StopOnError && result.IsFailure() {
				stopped.Store(true)
				mu.Lock()
				execResult.Stopped = true
				execResult.StopReason = "stopped on first error"
				mu.Unlock()
			}
		}(cmd)
	}

	wg.Wait()

	execResult.SortByOrder()
	execResult.TotalDuration = time.Since(startTime)
	return execResult
}

func executeSingleCommand(cmd RepoCommand, timeout time.Duration) Result {
	startTime := time.Now()

	stdout, stderr, err := executeCommand(cmd, timeout)

	result := Result{
		Command:  cmd,
		Success:  err == nil,
		Error:    err,
		Stdout:   stdout,
		Stderr:   stderr,
		Duration: time.Since(startTime),
		order:    cmd.order,
	}

	return result
}

// Skip mark a command result as skipped with a reason.
// This is used when a command is not executed due to some condition (e.g. no changes).
// See this command like a friendly way to indicate that a command was intentionally not run, rather than it being an error or failure.
// Return a Result with Success=false, Skipped=true, and the provided reason
func Skip(cmd RepoCommand, reason string) Result {
	return Result{
		Command:    cmd,
		Success:    false,
		Skipped:    true,
		SkipReason: reason,
		order:      cmd.order,
	}
}
