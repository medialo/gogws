package engine

type EventType int

const (
	EventJobStart EventType = iota
	EventLog
	EventErr
	EventJobEnd
)

func (e EventType) String() string {
	switch e {
	case EventJobStart:
		return "JOB_START"
	case EventLog:
		return "LOG"
	case EventErr:
		return "ERR"
	case EventJobEnd:
		return "JOB_END"
	default:
		return "UNKNOWN"
	}
}

type Event struct {
	GoroutineID int
	Type        EventType
	JobLabel    string
	Log         string
	Err         error
	Success     bool
}

func (e Event) IsStart() bool {
	return e.Type == EventJobStart
}

func (e Event) IsEnd() bool {
	return e.Type == EventJobEnd
}

func (e Event) IsLog() bool {
	return e.Type == EventLog
}

func (e Event) IsErr() bool {
	return e.Type == EventErr
}
