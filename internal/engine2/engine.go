package engine2

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

type EngineRender interface {
	//Total() int
	//Run() error
	Notify(ch <-chan EngineCommandEvent)
}

type ExecuteOptions struct {
	Parallel    int
	StopOnError bool
	Timeout     time.Duration
	Verbose     bool

	//OnStart func(cmd BaseCommand)
	//OnProgress func(current, total int, cmd BaseCommand)
	//OnComplete func(result Result)
}

func DefaultOptions() ExecuteOptions {
	return ExecuteOptions{
		Parallel:    5, // todo get from config
		StopOnError: false,
		Timeout:     0,
		Verbose:     false,
	}
}

func (opts ExecuteOptions) WithParallel(parallel int) *ExecuteOptions {
	opts.Parallel = parallel
	return &opts
}

func (opts ExecuteOptions) WithStopOnError(stopOnError bool) *ExecuteOptions {
	opts.StopOnError = stopOnError
	return &opts
}

func (opts ExecuteOptions) WithTimeout(timeout time.Duration) *ExecuteOptions {
	opts.Timeout = timeout
	return &opts
}

func (opts ExecuteOptions) WithVerbose(verbose bool) *ExecuteOptions {
	opts.Verbose = verbose
	return &opts
}

type Engine struct {
	taskChan chan func() error
	options  ExecuteOptions
}

func NewEngine(opts ExecuteOptions) Engine {
	return Engine{
		taskChan: make(chan func() error, 1),
		options:  opts,
	}
}

func (engine Engine) RunJobsAsync(task func() error) {
	engine.taskChan <- task
}

func notifyF(ch chan EngineCommandEvent, id int) EngineCommandNotify {
	return func(eventType EngineCommandEventType, log string) {
		ch <- EngineCommandEvent{
			id:        id,
			log:       log,
			eventType: eventType,
		}
	}
}

func (engine Engine) RunJobs(ctx context.Context, jobs []*EngineCommand) {
	nbGoRoutines := min(engine.options.Parallel, len(jobs))
	slog.Debug("Running jobs", "nbJobs", len(jobs), "nbGoRoutines", nbGoRoutines)
	jobsChan := make(chan EngineCommand, nbGoRoutines)
	ch := make(chan EngineCommandEvent, nbGoRoutines*len(jobs))
	wg := &sync.WaitGroup{}

	var jobsFinished atomic.Uint32
	jobsFinished.Store(0)

	for i := 0; i < nbGoRoutines; i++ {
		go func() {
			notify := notifyF(ch, i)
			notify(LOG, "idle")
			for job := range jobsChan {
				wg.Add(1)
				job(ctx, notify)
				jobsFinished.Add(1)
				wg.Done()
			}
		}()
	}

	go func() {
		for c := range ch {
			slog.Debug("Job event", "goRoutineId", c.id, "eventType", c.eventType, "log", c.log)
		}
	}()

	for _, job := range jobs {
		jobsChan <- *job
	}

	wg.Wait()
	close(jobsChan)
	close(ch)

	slog.Debug("Engine finished", "nbJobs", len(jobs), "nbJobsDone", jobsFinished.Load())
}

func (engine Engine) Close() {
	slog.Debug("Closing engine")
	close(engine.taskChan)
}
