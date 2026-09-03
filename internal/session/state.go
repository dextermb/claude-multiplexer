package session

type State int

const (
	StateStarting State = iota
	StateIdle
	StateBusy
	StateWaiting
	StateExited
	StateFailed
)

func (s State) String() string {
	switch s {
	case StateStarting:
		return "starting"
	case StateIdle:
		return "idle"
	case StateBusy:
		return "busy"
	case StateWaiting:
		return "waiting"
	case StateExited:
		return "exited"
	case StateFailed:
		return "failed"
	}
	return "unknown"
}

func (s State) Live() bool {
	return s == StateStarting || s == StateIdle || s == StateBusy || s == StateWaiting
}
