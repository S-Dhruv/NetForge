package job

type State string

const (
	Waiting State = "WAITING"
	Running State = "RUNNING"
	Stalled State = "STALLED"
	Failed State = "FAILED"
	Completed State = "COMPLETED"
)

func CanTransition(current State, next State) bool {
	switch current{
	case Waiting:
		return next == Running
	case Running:
		return next == Stalled || next == Failed || next == Completed
	case Stalled:
		return next == Running
	default:
		return false
	}
}


