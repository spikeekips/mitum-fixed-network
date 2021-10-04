package basicstates

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type HandoverState struct {
	*BaseConsensusState
	jivp *util.LockedItem
	af   *util.LockedItem
}

func NewHandoverState(
	db storage.Database,
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	pps *prprocessor.Processors,
) *HandoverState {
	bc := NewBaseConsensusState(base.StateHandover,
		db, policy, nodepool, suffrage, nil, pps)

	st := &HandoverState{
		BaseConsensusState: bc,
		jivp:               util.NewLockedItem(nil),
		af:                 util.NewLockedItem(false),
	}

	bc.broadcastACCEPTBallot = st.handoverBroadcastACCEPTBallot
	bc.broadcastNewINITBallot = st.handoverBroadcastNewINITBallot

	return st
}

func (st *HandoverState) Enter(sctx StateSwitchContext) (func() error, error) {
	if !st.suffrage.IsInside(st.nodepool.LocalNode().Address()) {
		st.Log().Error().Msg("local is not suffrage node; will move to consensus state")

		return nil, sctx.SetFromState(base.StateHandover).SetToState(base.StateConsensus)
	}

	if !st.underHandover() {
		st.Log().Debug().Msg("not under handover; will move to consensus state")

		return nil, sctx.SetFromState(base.StateHandover).SetToState(base.StateConsensus)
	}

	_ = st.jivp.Set(nil)
	_ = st.af.Set(false)

	callback, err := st.BaseConsensusState.Enter(sctx)
	if err != nil {
		return nil, err
	}

	return func() error {
		if err := callback(); err != nil {
			return err
		}

		defer func() {
			_ = st.af.Set(true)
		}()

		if st.States.isJoined() {
			st.Log().Debug().Msg("already joined")

			return nil
		}

		return st.States.joinDiscovery(1, nil)
	}, nil
}

func (st *HandoverState) Exit(sctx StateSwitchContext) (func() error, error) {
	callback, err := st.BaseConsensusState.Exit(sctx)
	if err != nil {
		return nil, err
	}

	_ = st.jivp.Set(nil)
	_ = st.af.Set(false)

	return func() error {
		if err := callback(); err != nil {
			return err
		}

		// NOTE if to state is consensus state, remove old node from passthrough
		if sctx.ToState() == base.StateConsensus {
			if err := st.finish(); err != nil {
				st.Log().Error().Err(err).Msg("failed to finish handover")

				return err
			}
		}

		return st.Timers().StopTimers([]localtime.TimerID{
			TimerIDBroadcastINITBallot,
			TimerIDFindProposal,
		})
	}, nil
}

func (st *HandoverState) ProcessVoteproof(voteproof base.Voteproof) error {
	if err := st.canMoveConsensus(voteproof); err != nil {
		return err // err type is StateSwitchContext
	}

	return st.BaseConsensusState.ProcessVoteproof(voteproof)
}

// passthroughFilter will block all the incoming message to old node, except
// current INIT after in handover(isInHandover() == true).
func (*HandoverState) passthroughFilter(voteproof base.Voteproof) func(network.PassthroughedSeal) bool {
	height := voteproof.Height()
	round := voteproof.Round()

	return func(psl network.PassthroughedSeal) bool {
		bl, ok := psl.Seal.(ballot.Ballot)
		if !ok {
			return true
		}

		if bl.Height() == height && bl.Round() == round && bl.Stage() == base.StageINIT {
			return true
		}

		return false
	}
}

func (st *HandoverState) joinedINITVoteproof() base.Voteproof {
	i := st.jivp.Value()
	if i == nil {
		return nil
	}

	return i.(base.Voteproof)
}

func (st *HandoverState) canMoveConsensus(voteproof base.Voteproof) error {
	jivp := st.joinedINITVoteproof()

	switch {
	case !st.afterJoining():
		return nil
	case !st.underHandover():
	case voteproof.Stage() != base.StageINIT:
		return nil
	case jivp == nil:
		st.whenNewVoteproofAfterJoin(voteproof)

		return nil
	case voteproof.Height() < jivp.Height():
		return nil
	case voteproof.Height() == jivp.Height():
		if voteproof.Round() <= jivp.Round() {
			return nil
		}
	}

	st.Log().Debug().Msg("received expected voteproof; moves to consensus")

	return st.NewStateSwitchContext(base.StateConsensus).
		SetVoteproof(voteproof)
}

func (st *HandoverState) whenNewVoteproofAfterJoin(voteproof base.Voteproof) {
	l := st.Log().With().Stringer("stage", voteproof.Stage()).
		Int64("height", voteproof.Height().Int64()).
		Uint64("round", voteproof.Round().Uint64()).
		Logger()

	_ = st.jivp.Set(voteproof)

	if on := st.oldNode(); on != nil {
		// NOTE update passthrough filter
		if err := st.nodepool.SetPassthrough(on, st.passthroughFilter(voteproof), 0); err != nil {
			st.Log().Error().Err(err).Msg("failed to update passthrough for old node with voteproof")

			return
		}

		l.Debug().Stringer("conninfo", on.ConnInfo()).Msg("update passthrough for old node with voteproof")
	}

	l.Debug().Msg("set voteproof after joining")
}

func (st *HandoverState) handoverBroadcastACCEPTBallot(
	newBlock,
	proposal valuehash.Hash,
	voteproof base.Voteproof,
	initialDelay time.Duration,
) error {
	if st.underHandover() {
		switch jivp := st.joinedINITVoteproof(); {
		case jivp == nil:
			return nil
		case base.CompareVoteproofSamePoint(jivp, voteproof) < 0:
			return nil
		}
	}

	st.Log().Debug().Stringer("stage", voteproof.Stage()).
		Int64("height", voteproof.Height().Int64()).
		Uint64("round", voteproof.Round().Uint64()).
		Msg("expected voteproof received for accept ballot")

	return st.BaseConsensusState.defaultBroadcastACCEPTBallot(newBlock, proposal, voteproof, initialDelay)
}

func (st *HandoverState) handoverBroadcastNewINITBallot(voteproof base.Voteproof) error {
	if st.underHandover() {
		switch jivp := st.joinedINITVoteproof(); {
		case jivp == nil:
			return nil
		case base.CompareVoteproofSamePoint(jivp, voteproof) < 0:
			return nil
		}
	}

	st.Log().Debug().Stringer("stage", voteproof.Stage()).
		Int64("height", voteproof.Height().Int64()).
		Uint64("round", voteproof.Round().Uint64()).
		Msg("expected voteproof received for new init ballot")

	return st.BaseConsensusState.defaultBroadcastNewINITBallot(voteproof)
}

func (st *HandoverState) finish() error {
	if !st.underHandover() {
		st.Log().Debug().Msg("trying to finish handover, but not under handover")

		return nil
	}

	st.Log().Debug().Msg("trying to finish handover")

	// NOTE requests *EndHandover* to old node
	hd := st.States.Handover().(*Handover)
	on := hd.OldNode()
	if on == nil {
		return nil
	}

	sl, err := hd.endHandoverSeal()
	if err != nil {
		return fmt.Errorf("failed to make EndHandoverSeal: %w", err)
	}

	switch ok, err := on.EndHandover(context.Background(), sl); {
	case err != nil:
		return fmt.Errorf("failed to send EndHandover seal: %w", err)
	case !ok:
		st.Log().Debug().Msg("old node said no in the response of EndHandover; ignore")
	}

	if err := hd.Stop(); err != nil {
		if !errors.Is(err, util.DaemonAlreadyStoppedError) {
			return fmt.Errorf("failed to stop handover: %w", err)
		}
	}

	st.Log().Debug().Bool("underhandover", st.underHandover()).Msg("handover stopped")

	if err := st.nodepool.RemovePassthrough(on.ConnInfo().String()); err != nil {
		if !errors.Is(err, util.NotFoundError) {
			st.Log().Error().Err(err).Msg("failed to remove passthrough of old node")

			return fmt.Errorf("failed to remove passthrough of old node: %w", err)
		}
	}

	return nil
}

func (st *HandoverState) afterJoining() bool {
	return st.af.Value().(bool)
}
