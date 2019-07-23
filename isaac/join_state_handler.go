package isaac

import (
	"context"
	"time"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/node"
	"golang.org/x/xerrors"
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
	*common.Logger
	*common.ReaderDaemon
	homeState                   *HomeState
	compiler                    *Compiler
	nt                          network.Network
	intervalBroadcastINITBallot time.Duration
	timeoutWaitVoteResult       time.Duration
	chanState                   chan node.State
	proposalChecker             *common.ChainChecker
	timer                       *common.CallbackTimer
	voteResultChecker           *common.ChainChecker
}

func NewJoinStateHandler(
	homeState *HomeState,
	compiler *Compiler,
	nt network.Network,
	intervalBroadcastINITBallot time.Duration,
	timeoutWaitVoteResult time.Duration,
) (*JoinStateHandler, error) {
	js := &JoinStateHandler{
		Logger:                      common.NewLogger(log, "module", "join-state-handler"),
		homeState:                   homeState,
		compiler:                    compiler,
		nt:                          nt,
		proposalChecker:             NewProposalCheckerJoin(homeState),
		intervalBroadcastINITBallot: intervalBroadcastINITBallot,
		timeoutWaitVoteResult:       timeoutWaitVoteResult,
	}
	js.ReaderDaemon = common.NewReaderDaemon(true, 0, nil)

	if homeState.PreviousBlock().Empty() {
		return nil, xerrors.Errorf("previous block is empty")
	}

	if intervalBroadcastINITBallot < time.Second*2 {
		js.Log().Warn("intervalBroadcastINITBallot is too short", "interval", intervalBroadcastINITBallot)
	}

	if timeoutWaitVoteResult <= intervalBroadcastINITBallot {
		js.Log().Warn("timeoutWaitVoteResult is too short", "timeout", timeoutWaitVoteResult)
	}

	js.voteResultChecker = NewJoinVoteResultChecker(homeState)

	return js, nil
}

func (js *JoinStateHandler) Start() error {
	if err := js.ReaderDaemon.Start(); err != nil {
		return err
	}

	if js.timer != nil {
		if err := js.timer.Stop(); err != nil {
			return err
		}
	}

	// NOTE keeps broadcasting init ballot, which is based on last block of
	// homeState until timeoutWaitVoteResult
	js.timer = common.NewCallbackTimer(
		"broadcasting-init-ballot-for-joining",
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
	if err := js.timer.Start(); err != nil {
		return err
	}

	go js.requestVoteProof()

	return nil
}

func (js *JoinStateHandler) Stop() error {
	if err := js.ReaderDaemon.Stop(); err != nil {
		return err
	}

	if !js.timer.IsStopped() {
		if err := js.timer.Stop(); err != nil {
			return err
		}
		js.timer = nil
	}

	return nil
}

func (js *JoinStateHandler) SetChanState(ch chan node.State) StateHandler {
	js.chanState = ch
	return js
}

func (js *JoinStateHandler) State() node.State {
	return node.StateBooting
}

func (js *JoinStateHandler) ReceiveProposal(proposal Proposal) error {
	err := js.proposalChecker.
		New(nil).
		SetContext(
			"proposal", proposal,
			"lastINITVoteResult", js.compiler.LastINITVoteResult(),
		).
		Check()

	if err != nil {
		return err
	}

	return nil
}

func (js *JoinStateHandler) ReceiveVoteResult(vr VoteResult) error {
	err := js.voteResultChecker.
		New(nil).
		SetContext(
			"vr", vr,
		).Check()
	if err != nil {
		return err
	}

	if !vr.GotMajority() {
		js.Log().Debug("got draw; ignore", "vr", vr)
		return nil
	}

	if vr.Stage() == StageINIT {
		if !js.timer.IsStopped() {
			_ = js.timer.Stop()
		}

		diff := vr.Height().Sub(js.homeState.Block().Height()).Int64()
		switch {
		case diff == 1: // network already stores 1 higher block
			// NOTE trying to catch up the latest vote result
			go js.catchUp(vr)
			return nil
		case diff == 0: // expected; move to consensus
			js.Log().Debug("got expected VoteResult; move to consensus", "vr", vr)
			js.chanState <- node.StateConsensus
			return nil
		case diff < 0: // something wrong, move to sync
			js.Log().Debug("got lower height VoteResult; move to sync", "vr", vr)
			js.chanState <- node.StateSync
			return nil
		default: // higher height received, move to sync
			js.Log().Debug("got higher height VoteResult; move to sync", "vr", vr)
			js.chanState <- node.StateSync
			return nil
		}
	}

	return nil
}

func (js *JoinStateHandler) broadcastINITBallot(common.Timer) error {
	ballot, err := NewINITBallot(
		js.homeState.Home().Address(),
		js.homeState.PreviousBlock().Hash(),
		js.homeState.Block().Height(),
		js.homeState.Block().Hash(),
		js.homeState.Block().Round(),
		js.homeState.Block().Proposal(),
	)
	if err != nil {
		return err
	}
	if err := ballot.Sign(js.homeState.Home().PrivateKey(), nil); err != nil {
		return err
	}

	if err := js.nt.Broadcast(ballot); err != nil {
		return err
	}

	js.Log().Debug("broadcast init ballot for joining", "ballot", ballot.Hash())
	return nil
}

func (js *JoinStateHandler) requestVoteProof() {
	//<-time.After(js.timeoutWaitVoteResult)

	if js.IsStopped() {
		return
	}

	js.Log().Debug("timeout to wait VoteResult; try to request VoteProof", "timeout", js.timeoutWaitVoteResult)

	if !js.timer.IsStopped() {
		if err := js.timer.Stop(); err != nil {
			js.Log().Error("failed to stop broadcastINITBallot timer", "error", err)
			return
		}
	}

	js.Log().Debug("trying to request VoteProof to suffrage members")

	sl, err := NewRequest(RequestVoteProof)
	if err != nil {
		js.Log().Error("failed to make vote proof request", "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	vps, err := js.nt.RequestAll(ctx, sl)
	if err != nil {
		js.Log().Error("failed to request vote proof request", "error", err)
		return
	}

	js.Log().Debug("got VoteProofs", "vote_proofs", vps)
}

func (js *JoinStateHandler) catchUp(vr VoteResult) {
	// TODO fix; it's just for testing
	block, err := NewBlock(vr.Height(), vr.Round(), vr.Proposal())
	if err != nil {
		js.Log().Error("failed to create new block from VoteResult", "vr", vr, "error", err)
		return
	}
	if !block.Hash().Equal(vr.Block()) {
		js.Log().Error(
			"new block from VoteResult does not match",
			"vr", vr,
			"block", block.Hash(),
			"vr_block", vr.Block(),
		)
		return
	}

	_ = js.homeState.SetBlock(block)

	js.Log().Debug("new block from VoteResult saved", "block", block)

	return
}
