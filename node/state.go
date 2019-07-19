package node

type State uint

const (
	StateBooting State = iota + 1
	StateJoin
	StateConsensus
	StateSync
	StateStopped
)

func (n State) IsValid() error {
	switch n {
	case StateBooting:
	case StateJoin:
	case StateConsensus:
	case StateSync:
	case StateStopped:
	default:
		return InvalidStateError
	}

	return nil
}

func (n State) String() string {
	switch n {
	case StateBooting:
		return "booting"
	case StateJoin:
		return "join"
	case StateConsensus:
		return "consensus"
	case StateSync:
		return "sync"
	case StateStopped:
		return "stopped"
	default:
		return "<empty node state>"
	}
}

func (n State) CanVote() bool {
	switch n {
	case StateJoin, StateConsensus:
		return true
	}

	return false
}
