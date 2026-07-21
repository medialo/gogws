package engine

import "time"

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
	if parallel < 1 {
		return opts
	}
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
