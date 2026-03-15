package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Options struct {
	Parallel    int
	StopOnError bool
	Timeout     time.Duration
}

func DefaultOptions() Options {
	return Options{
		Parallel:    5,
		StopOnError: false,
		Timeout:     0,
	}
}

func (opts Options) WithParallel(parallel int) Options {
	opts.Parallel = parallel
	return opts
}

func (opts Options) WithStopOnError(stopOnError bool) Options {
	opts.StopOnError = stopOnError
	return opts
}

func (opts Options) WithTimeout(timeout time.Duration) Options {
	opts.Timeout = timeout
	return opts
}

type Engine struct {
	options Options
}

func NewEngine(opts Options) *Engine {
	return &Engine{
		options: opts,
	}
}

type runningJob struct {
	job   Job
	index int
}

func (engine *Engine) RunJobs(ctx context.Context, jobs []Job) (<-chan Event, <-chan *ExecuteResult) {
	eventCh := make(chan Event, 100)
	resultCh := make(chan *ExecuteResult, 1)

	go engine.runJobsInternal(ctx, jobs, eventCh, resultCh)

	return eventCh, resultCh
}

func (engine *Engine) runJobsInternal(ctx context.Context, jobs []Job, eventCh chan Event, resultCh chan *ExecuteResult) {
	defer close(eventCh)
	defer close(resultCh)

	if len(jobs) == 0 {
		resultCh <- NewExecuteResult()
		return
	}

	nbWorkers := min(engine.options.Parallel, len(jobs))
	slog.Debug("Running jobs", "nbJobs", len(jobs), "nbWorkers", nbWorkers)

	jobsChan := make(chan runningJob, nbWorkers)
	resultsChan := make(chan Result, len(jobs))
	wg := &sync.WaitGroup{}

	var cancelCtx context.Context
	var cancel context.CancelFunc
	if engine.options.StopOnError {
		cancelCtx, cancel = context.WithCancel(ctx)
		defer cancel()
	} else {
		cancelCtx = ctx
		cancel = func() {}
	}

	if engine.options.Timeout > 0 {
		var timeoutCancel context.CancelFunc
		cancelCtx, timeoutCancel = context.WithTimeout(cancelCtx, engine.options.Timeout)
		defer timeoutCancel()
	}

	wg.Add(len(jobs))

	for workerID := 0; workerID < nbWorkers; workerID++ {
		go func(id int) {
			for rj := range jobsChan {
				select {
				case <-cancelCtx.Done():
					resultsChan <- Result{
						Label:      rj.job.Label,
						Success:    false,
						Skipped:    true,
						SkipReason: "cancelled",
						order:      rj.index,
					}
					wg.Done()
					continue
				default:
				}

				eventCh <- Event{
					GoroutineID: id,
					Type:        EventJobStart,
					JobLabel:    rj.job.Label,
				}

				start := time.Now()
				notify := func(eventType EventType, log string) {
					eventCh <- Event{
						GoroutineID: id,
						Type:        eventType,
						JobLabel:    rj.job.Label,
						Log:         log,
					}
				}

				err := rj.job.Fn(cancelCtx, notify)
				duration := time.Since(start)

				success := err == nil
				var jobErr error
				if !success {
					jobErr = err
					if engine.options.StopOnError {
						cancel()
					}
				}

				eventCh <- Event{
					GoroutineID: id,
					Type:        EventJobEnd,
					JobLabel:    rj.job.Label,
					Success:     success,
					Err:         jobErr,
				}

				resultsChan <- Result{
					Label:    rj.job.Label,
					Success:  success,
					Error:    jobErr,
					Duration: duration,
					order:    rj.index,
				}
				wg.Done()
			}
		}(workerID)
	}

	go func() {
		for i, job := range jobs {
			select {
			case <-cancelCtx.Done():
				for j := i; j < len(jobs); j++ {
					resultsChan <- Result{
						Label:      jobs[j].Label,
						Success:    false,
						Skipped:    true,
						SkipReason: "cancelled",
						order:      j,
					}
					wg.Done()
				}
				return
			case jobsChan <- runningJob{job: job, index: i}:
			}
		}
		close(jobsChan)
	}()

	wg.Wait()
	close(resultsChan)

	execResult := NewExecuteResult()
	for r := range resultsChan {
		execResult.AddResult(r)
	}
	execResult.SortByOrder()

	if cancelCtx.Err() != nil {
		execResult.Stopped = true
		execResult.StopReason = cancelCtx.Err().Error()
	}

	slog.Debug("Engine finished", "nbJobs", len(jobs), "success", execResult.SuccessCount(), "failed", execResult.FailedCount())
	resultCh <- execResult
}
