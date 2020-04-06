package isaac

import (
	"golang.org/x/xerrors"
)

type State uint8

const (
	_ State = iota
	// StateStopped indicates node is in state, node process is
	// finished.
	StateStopped
	// StateBooting indicates node is in state, node checks it's state.
	StateBooting
	// StateJoining indicates node is in state, node is trying to
	// join network.
	StateJoining
	// StateConsensus indicates node is in state, node participates
	// consensus with the other nodes.
	StateConsensus
	// StateSyncing indicates node is in state, node is syncing block.
	StateSyncing
	// StateBroken indicates that node can not participates network
	// with various kind of reason.
	StateBroken
)

func (st State) String() string {
	switch st {
	case StateStopped:
		return "STOPPED"
	case StateBooting:
		return "BOOTING"
	case StateJoining:
		return "JOINING"
	case StateConsensus:
		return "CONSENSUS"
	case StateSyncing:
		return "SYNCING"
	case StateBroken:
		return "BROKEN"
	default:
		return "<unknown State>"
	}
}

func (st State) IsValid([]byte) error {
	switch st {
	case StateStopped, StateBooting, StateJoining, StateConsensus, StateSyncing, StateBroken:
		return nil
	}

	return xerrors.Errorf("invalid state found; state=%d", st)
}

func (st State) MarshalText() ([]byte, error) {
	return []byte(st.String()), nil
}
