package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

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

// RunJobs runs the given jobs in parallel.
// It returns a channel of events and a channel of execution results.
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
		resultCh <- NewNoExecutionResult()
		return
	}

	nbWorkers := min(engine.options.Parallel, len(jobs))
	slog.Debug("Running jobs", "nbJobs", len(jobs), "nbWorkers", nbWorkers)

	jobsChan := make(chan runningJob, nbWorkers)
	//resultsChan := make(chan JobResult, len(jobs))
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

	execResult := NewExecuteResult(len(jobs))

	// launch x go routine worker
	for workerID := 0; workerID < nbWorkers; workerID++ {
		go func(id int) {
			for runJob := range jobsChan { // worker get job from jobsChan
				select {
				case <-cancelCtx.Done():
					//resultsChan <- JobResult{
					//	JobId:      runJob.job.JobNameId,
					//	Success:    false,
					//	Skipped:    true,
					//	SkipReason: "cancelled",
					//	order:      runJob.index,
					//}
					execResult.AddResult(JobResult{
						JobId:      runJob.job.JobNameId,
						Success:    false,
						Skipped:    true,
						SkipReason: "cancelled",
						order:      runJob.index,
					})
					wg.Done()
					continue

				default:
				}

				eventCh <- Event{
					GoroutineID: id,
					Type:        EventJobStart,
					JobNameId:   runJob.job.JobNameId,
				}

				start := time.Now()

				notify := func(eventType EventType, log string) {
					eventCh <- Event{
						GoroutineID: id,
						Type:        eventType,
						JobNameId:   runJob.job.JobNameId,
						Log:         log,
					}
				}

				err := runJob.job.Fn(cancelCtx, notify)
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
					JobNameId:   runJob.job.JobNameId,
					Success:     success,
					Err:         jobErr,
				}

				//resultsChan <- JobResult{
				//	JobId:    runJob.job.JobNameId,
				//	Success:  success,
				//	Error:    jobErr,
				//	Duration: duration,
				//	order:    runJob.index,
				//}
				execResult.AddResult(JobResult{
					JobId:    runJob.job.JobNameId,
					Success:  success,
					Error:    jobErr,
					Duration: duration,
					order:    runJob.index,
				})
				wg.Done()
			}
		}(workerID)
	}

	//go func() {
	//	for i, job := range jobs {
	//		jobsChan <- runningJob{job: job, index: i}
	//	}
	//	close(jobsChan)
	//}()

	for i, job := range jobs {
		jobsChan <- runningJob{job: job, index: i}
	}
	close(jobsChan)

	wg.Wait()
	//close(resultsChan)

	//for r := range resultsChan {
	//	execResult.AddResult(r)
	//}
	execResult.SortByOrder()

	if cancelCtx.Err() != nil {
		execResult.Stopped = true
		execResult.StopReason = cancelCtx.Err().Error()
	}

	slog.Debug("Engine finished", "nbJobs", len(jobs), "success", execResult.SuccessCount(), "failed", execResult.FailedCount())
	resultCh <- execResult
}
