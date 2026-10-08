package engine

type Event interface {
	JobID() string
	WorkerID() int
	isEvent()
}

type eventBase struct {
	Worker int
	Job    string
}

func (b eventBase) JobID() string { return b.Job }

func (b eventBase) WorkerID() int { return b.Worker }

func (eventBase) isEvent() {}

type JobStarted struct {
	eventBase
}

type JobLog struct {
	eventBase
	Line string
}

type JobPhase struct {
	eventBase
	Phase string
}

type JobProgress struct {
	eventBase
	Progress
}

type JobEnded struct {
	eventBase
	Success bool
	Err     error
}
