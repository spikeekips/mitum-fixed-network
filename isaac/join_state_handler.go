package isaac

import (
	"context"
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/node"
)

// JoinStateHandler tries to join network safely. This is basic strategy,
// * keeps broadcasting init ballot, which is based on last block of homeState
// * wait INIT VoteResult of next block
// 	- next init VoteResult is expected by last block of homeState, move to
// 	consensus
// 	- next init VoteResult is higher by 1 height from last block of homeState,
// 	catch up the result and wait next init
// 	- next init VoteResult is too high rather than homeState, move to sync
// * failed to get INIT VoteResult of next block
// 	- requests VoteProof to suffrage members
// 	- select the valid and highest VoteProof
// 	- broadcast INIT ballot, based on highest VoteProof
type JoinStateHandler struct {
	sync.RWMutex
	*common.Logger
	homeState                   *HomeState
	compiler                    *Compiler
	nt                          network.Network
	suffrage                    Suffrage
	ballotMaker                 BallotMaker
	proposalValidator           ProposalValidator
	intervalBroadcastINITBallot time.Duration
	timeoutWaitVoteResult       time.Duration
	chanState                   chan StateContext
	started                     bool
	timer                       *common.CallbackTimer
	proposalChecker             *common.ChainChecker
	voteResultChecker           *common.ChainChecker
}

func NewJoinStateHandler(
	homeState *HomeState,
	compiler *Compiler,
	nt network.Network,
	suffrage Suffrage,
	ballotMaker BallotMaker,
	proposalValidator ProposalValidator,
	intervalBroadcastINITBallot time.Duration,
	timeoutWaitVoteResult time.Duration,
) (*JoinStateHandler, error) {
	if homeState.PreviousBlock().Empty() {
		return nil, xerrors.Errorf("previous block is empty")
	}

	return &JoinStateHandler{
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "s.h.join")
		}),
		homeState:                   homeState,
		compiler:                    compiler,
		nt:                          nt,
		suffrage:                    suffrage,
		ballotMaker:                 ballotMaker,
		proposalValidator:           proposalValidator,
		intervalBroadcastINITBallot: intervalBroadcastINITBallot,
		timeoutWaitVoteResult:       timeoutWaitVoteResult,
		proposalChecker:             NewProposalCheckerJoin(homeState, suffrage),
		voteResultChecker:           NewJoinVoteResultChecker(homeState),
	}, nil
}

func (js *JoinStateHandler) Start() error {
	if js.intervalBroadcastINITBallot < time.Second*2 {
		js.Log().Warn().Dur("interval", js.intervalBroadcastINITBallot).Msg("intervalBroadcastINITBallot is too short")
	}

	if js.timeoutWaitVoteResult <= js.intervalBroadcastINITBallot {
		js.Log().Warn().Dur("timeout", js.timeoutWaitVoteResult).Msg("timeoutWaitVoteResult is too short")
	}

	_ = js.Stop() // nolint

	js.Lock()
	defer js.Unlock()
	js.started = true

	return nil
}

func (js *JoinStateHandler) Stop() error {
	if err := js.Deactivate(); err != nil {
		return err
	}

	js.Lock()
	defer js.Unlock()
	js.started = false

	return nil
}

func (js *JoinStateHandler) IsStopped() bool {
	js.RLock()
	defer js.RUnlock()

	return !js.started
}

func (js *JoinStateHandler) Activate(StateContext) error {
	_ = js.stopTimer() // nolint

	js.Lock()
	defer js.Unlock()

	// NOTE keeps broadcasting init ballot, which is based on last block of
	// homeState until timeoutWaitVoteResult
	js.timer = common.NewCallbackTimer(
		"join-broadcasting-init",
		js.intervalBroadcastINITBallot,
		js.broadcastINITBallot,
	).
		SetIntervalFunc(func(runCount uint, elapsed time.Duration) time.Duration {
			if runCount < 1 { // this makes to broadcast without waiting
				return time.Nanosecond
			}

			if elapsed > js.timeoutWaitVoteResult {
				go js.requestVoteProof()
				return 0
			}

			return js.intervalBroadcastINITBallot
		})
	js.timer.SetLogger(*js.Log())

	if err := js.timer.Start(); err != nil {
		return err
	}

	return nil
}

func (js *JoinStateHandler) Deactivate() error {
	return js.stopTimer()
}

func (js *JoinStateHandler) SetChanState(ch chan StateContext) StateHandler {
	js.chanState = ch
	return js
}

func (js *JoinStateHandler) State() node.State {
	return node.StateBooting
}

func (js *JoinStateHandler) stopTimer() error {
	js.RLock()
	defer js.RUnlock()

	if js.timer == nil || js.timer.IsStopped() {
		return nil
	}

	if err := js.timer.Stop(); err != nil {
		js.Log().Error().Err(err).Msg("failed to stop timer")
		return err
	}

	return nil
}

func (js *JoinStateHandler) ReceiveProposal(proposal Proposal) error {
	err := js.proposalChecker.
		New(context.TODO()).
		SetContext("proposal", proposal).
		SetContext("lastINITVoteResult", js.compiler.LastINITVoteResult()).
		Check()
	if err != nil {
		return err
	}

	return nil
}

func (js *JoinStateHandler) ReceiveVoteResult(vr VoteResult) error {
	err := js.voteResultChecker.
		New(context.TODO()).
		SetContext("vr", vr).
		SetContext("lastINITVoteResult", js.compiler.LastINITVoteResult()).
		Check()
	if err != nil {
		return err
	}

	js.Log().Debug().Object("vr", vr).Msg("VoteResult checked")

	if !vr.GotMajority() {
		js.Log().Debug().Object("vr", vr).Msg("got not majority; ignore")
		return nil
	}

	if vr.Stage() == StageINIT {
		return js.gotINITMajority(vr)
	} else {
		return js.gotNotINITMajority(vr)
	}
}

func (js *JoinStateHandler) broadcastINITBallot(common.Timer) error {
	ballot, err := js.ballotMaker.INIT(
		js.homeState.PreviousBlock().Hash(),
		js.homeState.Block().Round(),
		js.homeState.Block().Height().Add(1),
		js.homeState.Block().Hash(),
		Round(0),
		js.homeState.Block().Proposal(),
	)
	if err != nil {
		return err
	}

	js.Log().Debug().Object("ballot", ballot.Hash()).Msg("broadcast init ballot for joining")
	if err := js.nt.Broadcast(ballot); err != nil {
		return err
	}

	return nil
}

func (js *JoinStateHandler) requestVoteProof() {
	//<-time.After(js.timeoutWaitVoteResult)

	if js.IsStopped() {
		return
	}

	js.Log().Debug().Dur("timeout", js.timeoutWaitVoteResult).Msg("timeout to wait VoteResult; try to request VoteProof")

	if err := js.stopTimer(); err != nil {
		return
	}

	js.Log().Debug().Msg("trying to request VoteProof to suffrage members")

	sl, err := NewRequest(RequestVoteProof)
	if err != nil {
		js.Log().Error().Err(err).Msg("failed to make vote proof request")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	vps, err := js.nt.RequestAll(ctx, sl)
	if err != nil {
		js.Log().Error().Err(err).Msg("failed to request vote proof request")
		return
	}

	if js.Log().Debug().Enabled() {
		lvps := zerolog.Dict()
		for k, v := range vps {
			lvps = lvps.Object(k.String(), v)
		}

		js.Log().Debug().Dict("vote_proofs", lvps).Msg("got VoteProofs")
	}
}

func (js *JoinStateHandler) catchUp(vr VoteResult) error {
	if !js.homeState.Block().Hash().Equal(vr.LastBlock()) {
		return xerrors.Errorf(
			"block hash does not match",
			"current", js.homeState.Block().Hash(),
			"last_block_vr", vr.LastBlock(),
		)
	}

	block, err := js.proposalValidator.NewBlock(vr.Proposal())
	if err != nil {
		js.Log().Error().Err(err).Object("vr", vr).Msg("failed to make new block from proposal")
		return err
	}

	_ = js.homeState.SetBlock(block)

	js.Log().Debug().Object("block", block).Msg("new block from VoteResult saved")

	return nil
}

func (js *JoinStateHandler) gotINITMajority(vr VoteResult) error {
	_ = js.stopTimer() // nolint

	diff := vr.Height().Sub(js.homeState.Block().Height()).Int64()
	switch {
	case diff == 2: // network already stores 1 higher block
		// NOTE trying to catch up the latest vote result
		go func() {
			if err := js.catchUp(vr); err != nil {
				js.Log().Error().Err(err).Msg("failed to catchup")
			}
		}()

		return nil
	case diff == 1: // expected; move to consensus
		js.Log().Debug().Object("vr", vr).Msg("got expected VoteResult; move to consensus")
		js.chanState <- NewStateContext(node.StateConsensus).
			SetContext("vr", vr)
		return nil
	case diff < 0: // something wrong, move to sync
		js.Log().Debug().Object("vr", vr).Msg("got lower height VoteResult; move to sync")
		js.chanState <- NewStateContext(node.StateSyncing).
			SetContext("vr", vr)
		return nil
	default: // higher height received, move to sync
		js.Log().Debug().Object("vr", vr).Msg("got higher height VoteResult; move to sync")
		js.chanState <- NewStateContext(node.StateSyncing).
			SetContext("vr", vr)
		return nil
	}
}

func (js *JoinStateHandler) gotNotINITMajority(vr VoteResult) error {
	js.Log().Debug().Object("vr", vr).Msg("got majority, not init; will be ignored")
	return nil
}
