package basicstates

import (
	"context"
	"fmt"

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
	ctx       context.Context
	ae        bool
}

func NewStateSwitchContext(from, to base.State) StateSwitchContext {
	return StateSwitchContext{
		to:   to,
		from: from,
		ctx:  context.Background(),
	}
}

func (sctx StateSwitchContext) IsValid([]byte) error {
	if !sctx.ae {
		if err := sctx.from.IsValid(nil); err != nil {
			return err
		}
	}

	return isvalid.Check(nil, false, sctx.to)
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

func (sctx StateSwitchContext) allowEmpty(b bool) StateSwitchContext {
	sctx.ae = b

	return sctx
}

func (sctx StateSwitchContext) Context() context.Context {
	return sctx.ctx
}

func (sctx StateSwitchContext) SetContext(ctx context.Context) StateSwitchContext {
	sctx.ctx = ctx

	return sctx
}

func (sctx StateSwitchContext) Return() StateSwitchContext {
	to := sctx.to
	from := sctx.from

	return sctx.SetToState(from).SetFromState(to)
}

func (sctx StateSwitchContext) ContextWithValue(k util.ContextKey, v interface{}) StateSwitchContext {
	sctx.ctx = context.WithValue(sctx.ctx, k, v)

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
	return sctx.to == base.StateEmpty
}
