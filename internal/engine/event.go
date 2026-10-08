package engine

type EventType int

const (
	EventJobStart EventType = iota
	EventJobLog             // this event will be used to display string at right of job
	EventJobErr
	EventJobEnd
	EventJobSkipped
	EventSlog // todo fusionner le slog et le log ?
	EventJobPhase
	EventJobProgress
)

func (e EventType) String() string {
	switch e {
	case EventJobStart:
		return "JOB_START"
	case EventJobLog:
		return "LOG"
	case EventJobErr:
		return "ERR"
	case EventJobEnd:
		return "JOB_END"
	case EventJobSkipped:
		return "JOB_SKIPPED"
	case EventJobPhase:
		return "JOB_PHASE"
	case EventJobProgress:
		return "JOB_PROGRESS"
	default:
		return "UNKNOWN"
	}
}

type Event struct {
	GoroutineID int
	Type        EventType
	JobNameId   string
	Log         string
	Err         error
	Success     bool
	Progress    *Progress
}

func (e Event) IsStart() bool {
	return e.Type == EventJobStart
}

func (e Event) IsEnd() bool {
	return e.Type == EventJobEnd
}

func (e Event) IsLog() bool {
	return e.Type == EventJobLog
}

func (e Event) IsErr() bool {
	return e.Type == EventJobErr
}
