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

// ParseState is the inverse of String, so a state that crossed the peer stream
// as a word reads back to a State. An unknown word reads as StateStarting.
func ParseState(word string) State {
	switch word {
	case "idle":
		return StateIdle
	case "busy":
		return StateBusy
	case "waiting":
		return StateWaiting
	case "exited":
		return StateExited
	case "failed":
		return StateFailed
	default:
		return StateStarting
	}
}
