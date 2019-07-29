package isaac

import (
	"context"
	"reflect"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/node"
)

type StateHandler interface {
	common.Daemon
	Activate(StateContext) error
	Deactivate() error
	State() node.State
	SetChanState(chan StateContext) StateHandler
	ReceiveVoteResult(VoteResult) error
	ReceiveProposal(Proposal) error
}

type StateContext struct {
	state node.State
	ctx   context.Context
}

func NewStateContext(state node.State) StateContext {
	return StateContext{state: state, ctx: context.Background()}
}

func (sc StateContext) State() node.State {
	return sc.state
}

func (sc StateContext) Context() context.Context {
	return sc.ctx
}

func (sc StateContext) SetContext(key, value interface{}) StateContext {
	if sc.ctx == nil {
		sc.ctx = context.Background()
	}

	sc.ctx = context.WithValue(sc.ctx, key, value)
	return sc
}

func (sc StateContext) ContextValue(key interface{}, value interface{}) error {
	if sc.ctx == nil {
		return common.ContextValueNotFoundError.Newf("key='%v'", key)
	}

	v := sc.ctx.Value(key)
	if v == nil {
		return common.ContextValueNotFoundError.Newf("key='%v'", key)
	}

	reflect.ValueOf(value).Elem().Set(reflect.ValueOf(v))

	return nil
}
