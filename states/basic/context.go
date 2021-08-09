package basicstates

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/isvalid"
)

var SyncByVoteproofError = util.NewError("sync by voteproof")

type StateSwitchContext struct {
	to        base.State
	from      base.State
	voteproof base.Voteproof
	err       error
}

func NewStateSwitchContext(from, to base.State) StateSwitchContext {
	return StateSwitchContext{
		to:   to,
		from: from,
	}
}

func (sctx StateSwitchContext) IsValid([]byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		sctx.to,
		sctx.from,
	}, nil, false); err != nil {
		return err
	}

	if sctx.to == base.StateUnknown || sctx.from == base.StateUnknown {
		return errors.Errorf("invalid state, to=%v from=%v", sctx.to, sctx.from)
	}

	return nil
}

func (sctx StateSwitchContext) Voteproof() base.Voteproof {
	return sctx.voteproof
}

func (sctx StateSwitchContext) SetVoteproof(voteproof base.Voteproof) StateSwitchContext {
	sctx.voteproof = voteproof

	return sctx
}

func (sctx StateSwitchContext) Err() error {
	return sctx.err
}

func (sctx StateSwitchContext) Error() string {
	if sctx.err == nil {
		return fmt.Sprintf("<StateSwitchContext: %v -> %v>", sctx.from, sctx.to)
	}

	return sctx.err.Error()
}

func (sctx StateSwitchContext) SetError(err error) StateSwitchContext {
	sctx.err = err

	return sctx
}

func (sctx StateSwitchContext) FromState() base.State {
	return sctx.from
}

func (sctx StateSwitchContext) SetFromState(st base.State) StateSwitchContext {
	sctx.from = st

	return sctx
}

func (sctx StateSwitchContext) ToState() base.State {
	return sctx.to
}

func (sctx StateSwitchContext) SetToState(st base.State) StateSwitchContext {
	sctx.to = st

	return sctx
}

func (sctx StateSwitchContext) MarshalText() ([]byte, error) {
	return jsonenc.Marshal(map[string]interface{}{
		"to":        sctx.to,
		"from":      sctx.from,
		"error":     sctx.err,
		"voteproof": sctx.voteproof,
	})
}

func (sctx StateSwitchContext) IsEmpty() bool {
	return sctx.to == base.StateUnknown
}
