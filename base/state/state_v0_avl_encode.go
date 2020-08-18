package state

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func (stav *StateV0AVLNode) unpack(
	enc encoder.Encoder,
	bHash []byte,
	height int16,
	bLeft,
	bLeftHash,
	bRight,
	bRightHash,
	bState []byte,
) error {
	var state StateV0
	if s, err := DecodeState(enc, bState); err != nil {
		return err
	} else if sv, ok := s.(StateV0); !ok {
		return hint.InvalidTypeError.Errorf("not state.StateV0; type=%T", s)
	} else {
		state = sv
	}

	stav.height = height
	stav.h = bHash
	stav.left = bLeft
	stav.leftHash = bLeftHash
	stav.right = bRight
	stav.rightHash = bRightHash
	stav.state = state

	return nil
}
