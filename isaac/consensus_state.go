package isaac

import (
	"github.com/spikeekips/mitum/errors"
)

var (
	InvalidConsensusStateError = errors.NewError("invalid ConsensusState")
)

type ConsensusState uint8

const (
	_ ConsensusState = iota
	// ConsensusStateStopped indicates node is in state, node process is
	// finished.
	ConsensusStateStopped
	// ConsensusStateBooting indicates node is in state, node checks it's state.
	ConsensusStateBooting
	// ConsensusStateJoining indicates node is in state, node is trying to
	// join network.
	ConsensusStateJoining
	// ConsensusStateConsensus indicates node is in state, node participates
	// consensus with the other nodes.
	ConsensusStateConsensus
	// ConsensusStateSyncing indicates node is in state, node is syncing block.
	ConsensusStateSyncing
	// ConsensusStateBroken indicates that node can not participates network
	// with various kind of reason.
	ConsensusStateBroken
)

func (st ConsensusState) String() string {
	switch st {
	case ConsensusStateStopped:
		return "STOPPED"
	case ConsensusStateBooting:
		return "BOOTING"
	case ConsensusStateJoining:
		return "JOINING"
	case ConsensusStateConsensus:
		return "CONSENSUS"
	case ConsensusStateSyncing:
		return "SYNCING"
	case ConsensusStateBroken:
		return "BROKEN"
	default:
		return "<unknown ConsensusState>"
	}
}

func (st ConsensusState) IsValid([]byte) error {
	switch st {
	case ConsensusStateStopped, ConsensusStateBooting, ConsensusStateJoining,
		ConsensusStateConsensus, ConsensusStateSyncing, ConsensusStateBroken:
		return nil
	}

	return InvalidConsensusStateError.Wrapf("ConsensusState=%d", st)
}

func (st ConsensusState) MarshalText() ([]byte, error) {
	return []byte(st.String()), nil
}
