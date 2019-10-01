package node

type State uint

const (
	StateBooting State = iota + 1
	StateJoining
	StateConsensus
	StateSyncing
	StateStopped
)

func (n State) IsValid() error {
	switch n {
	case StateBooting:
	case StateJoining:
	case StateConsensus:
	case StateSyncing:
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
	case StateJoining:
		return "joining"
	case StateConsensus:
		return "consensus"
	case StateSyncing:
		return "syncing"
	case StateStopped:
		return "stopped"
	default:
		return "<empty node state>"
	}
}

func (n State) CanVote() bool {
	switch n {
	case StateJoining, StateConsensus:
		return true
	}

	return false
}
