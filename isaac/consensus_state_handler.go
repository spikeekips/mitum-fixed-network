package isaac

import (
	"context"
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/node"
)

// ConsensusStateHandler is only for suffrage node
type ConsensusStateHandler struct {
	sync.RWMutex
	*common.Logger
	homeState         *HomeState
	compiler          *Compiler
	nt                network.Network
	suffrage          Suffrage
	proposalValidator ProposalValidator
	proposalMaker     ProposalMaker
	timeoutWaitBallot time.Duration
	started           bool
	chanState         chan StateContext
	timer             *common.CallbackTimer
	proposalChecker   *common.ChainChecker
	voteResultChecker *common.ChainChecker
}

func NewConsensusStateHandler(
	homeState *HomeState,
	compiler *Compiler,
	nt network.Network,
	suffrage Suffrage,
	proposalValidator ProposalValidator,
	proposalMaker ProposalMaker,
	timeoutWaitBallot time.Duration,
) (*ConsensusStateHandler, error) {
	if homeState.PreviousBlock().Empty() {
		return nil, xerrors.Errorf("previous block is empty")
	}

	logger := common.NewLogger(log, "module", "consensus-state-handler")

	if timeoutWaitBallot < time.Nanosecond {
		logger.Log().Warn("timeoutWaitBallot is too short", "timeout", timeoutWaitBallot)
	}

	return &ConsensusStateHandler{
		Logger:            logger,
		homeState:         homeState,
		compiler:          compiler,
		nt:                nt,
		suffrage:          suffrage,
		proposalValidator: proposalValidator,
		proposalMaker:     proposalMaker,
		timeoutWaitBallot: timeoutWaitBallot,
		proposalChecker:   NewProposalCheckerConsensus(homeState, suffrage),
		voteResultChecker: NewConsensusVoteResultChecker(homeState),
	}, nil
}

func (cs *ConsensusStateHandler) Start() error {
	_ = cs.Stop() // nolint

	cs.Lock()
	defer cs.Unlock()
	cs.started = true

	return nil
}

func (cs *ConsensusStateHandler) Stop() error {
	if err := cs.Deactivate(); err != nil {
		return err
	}

	cs.Lock()
	defer cs.Unlock()
	cs.started = false

	return nil
}

func (cs *ConsensusStateHandler) IsStopped() bool {
	cs.RLock()
	defer cs.RUnlock()

	return !cs.started
}

func (cs *ConsensusStateHandler) Activate(sct StateContext) error {
	_ = cs.stopTimer() // nolint

	var vr VoteResult
	if err := sct.ContextValue("vr", &vr); err != nil {
		return xerrors.Errorf("ConsensusStateHandler fail to Activate(); %w", err)
	}

	if vr.Stage() != StageINIT {
		return xerrors.Errorf("stage of activated VoteResult should be StageINIT; stage=%q", vr.Stage())
	}

	go func() {
		// propose proposal with VoteResult
		if err := cs.prepareProposal(vr); err != nil {
			cs.Log().Error("failed to propose", "error", err)
		}
	}()

	return nil
}

func (cs *ConsensusStateHandler) Deactivate() error {
	return cs.stopTimer()
}

func (cs *ConsensusStateHandler) SetChanState(ch chan StateContext) StateHandler {
	cs.chanState = ch
	return cs
}

func (cs *ConsensusStateHandler) State() node.State {
	return node.StateConsensus
}

func (cs *ConsensusStateHandler) stopTimer() error {
	cs.RLock()
	defer cs.RUnlock()

	if cs.timer == nil || cs.timer.IsStopped() {
		return nil
	}

	if err := cs.timer.Stop(); err != nil {
		cs.Log().Error("failed to stop timer", "error", err)
		return err
	}

	return nil
}

func (cs *ConsensusStateHandler) ReceiveProposal(proposal Proposal) error {
	err := cs.proposalChecker.
		New(context.TODO()).
		SetContext("proposal", proposal).
		SetContext("lastINITVoteResult", cs.compiler.LastINITVoteResult()).
		Check()
	if err != nil {
		return err
	}

	cs.Log().Debug("proposal checked", "proposal", proposal.Hash())

	err = cs.nextRoundTimer(
		"wait-ballot-timeout-next-round-consensus",
		cs.compiler.LastINITVoteResult(),
	)
	if err != nil {
		return err
	}

	// TODO validate proposal
	var block Block
	if block, err = cs.proposalValidator.NewBlock(
		proposal.Height(),
		proposal.Round(),
		proposal.Hash(),
	); err != nil {
		return err
	}

	acting := cs.suffrage.Acting(proposal.Height(), proposal.Round())
	insideActing := acting.Exists(cs.homeState.Home().Address())
	if !insideActing {
		cs.Log().Debug(
			"not acting member at this proposal; not broadcast sign ballot",
			"proposal", proposal.Hash(),
			"height", proposal.Height(),
			"round", proposal.Round(),
		)
	} else {
		cs.Log().Debug(
			"acting member at this proposal; broadcast sign ballot",
			"proposal", proposal.Hash(),
			"height", proposal.Height(),
			"round", proposal.Round(),
		)
	}

	if insideActing {
		ballot, err := NewSIGNBallot(
			cs.homeState.Home().Address(),
			cs.homeState.Block().Hash(),
			cs.homeState.Block().Round(),
			block.Height(),
			block.Hash(),
			block.Round(),
			block.Proposal(),
		)
		if err != nil {
			return err
		}
		if err := ballot.Sign(cs.homeState.Home().PrivateKey(), nil); err != nil {
			return err
		}

		if err := cs.nt.Broadcast(ballot); err != nil {
			return err
		}
	}

	return nil
}

func (cs *ConsensusStateHandler) ReceiveVoteResult(vr VoteResult) error {
	err := cs.voteResultChecker.
		New(context.TODO()).
		SetContext("vr", vr).
		SetContext("lastINITVoteResult", cs.compiler.LastINITVoteResult()).
		Check()
	if err != nil {
		return err
	}

	cs.Log().Debug("VoteResult checked", "vr", vr)

	if vr.GotDraw() {
		return cs.startNextRound(vr)
	} else if vr.GotMajority() {
		if vr.Stage() == StageINIT {
			return cs.gotINITMajority(vr)
		} else {
			return cs.gotNotINITMajority(vr)
		}
	}

	return nil
}

func (cs *ConsensusStateHandler) gotINITMajority(vr VoteResult) error {
	_ = cs.stopTimer() // nolint

	diff := vr.Height().Sub(cs.homeState.Block().Height()).Int64()
	switch {
	case diff == 2: // move to next block
		cs.Log().Debug("got VoteResult of next block; keep going", "vr", vr)

		if !cs.homeState.Block().Hash().Equal(vr.LastBlock()) {
			cs.Log().Error(
				"init for next block; last block does not match; move to sync",
				"home", cs.homeState.Block().Hash(),
				"block_vr", vr.LastBlock(),
				"vr", vr,
			)
			cs.chanState <- NewStateContext(node.StateSync).
				SetContext("vr", vr)

			return xerrors.Errorf("init for next block; last block does not match; move to sync")
		}

		// TODO store new block; fix; it's just for testing
		block, err := NewBlockFromVoteResult(vr)
		if err != nil {
			cs.Log().Error("failed to create new block from VoteResult", "vr", vr, "error", err)
			return err
		}

		_ = cs.homeState.SetBlock(block)

		cs.Log().Info("new block created", "block", block, "vr", vr)
	case diff == 1: // next round
		cs.Log().Debug("got VoteResult of next round; keep going", "vr", vr)

		if !cs.homeState.Block().Hash().Equal(vr.Block()) {
			cs.Log().Error(
				"init for next round; block does not match; move to sync",
				"home", cs.homeState.Block().Hash(),
				"block_vr", vr.Block(),
				"vr", vr,
			)
			cs.chanState <- NewStateContext(node.StateSync).
				SetContext("vr", vr)

			return xerrors.Errorf("init for next round; block does not match; move to sync")
		}
	default: // unexpected height received, move to sync
		cs.Log().Debug("got not expected height VoteResult; move to sync", "vr", vr)
		cs.chanState <- NewStateContext(node.StateSync).
			SetContext("vr", vr)
		return xerrors.Errorf("got not expected height VoteResult; move to sync")
	}

	return cs.prepareProposal(vr)
}

func (cs *ConsensusStateHandler) gotNotINITMajority(vr VoteResult) error {
	// TODO check NilHash
	// TODO broadcast next stage ballot

	switch vr.Stage() {
	case StageSIGN, StageACCEPT:
	default:
		return xerrors.Errorf("invalid stage found", "vr", vr)
	}

	if !cs.proposalValidator.Validated(vr.Proposal()) {
		cs.Log().Debug("proposal did not validated; validate it", "vr", vr)
	}

	block, err := cs.proposalValidator.NewBlock(vr.Height(), vr.Round(), vr.Proposal())
	if err != nil {
		cs.Log().Error("failed to make new block from proposal", "vr", vr, "error", err)
		return err
	}

	if !vr.Block().Equal(block.Hash()) {
		cs.Log().Warn(
			"block hash does not match",
			"vr_block", vr.Block(),
			"block", block.Hash(),
			"vr", vr,
		)
	}

	var ballot Ballot
	switch vr.Stage() {
	case StageSIGN:
		acting := cs.suffrage.Acting(vr.Height(), vr.Round())
		if !acting.Exists(cs.homeState.Home().Address()) {
			cs.Log().Debug(
				"not acting member at this VoteResult; not broadcast accept ballot",
				"vr", vr,
				"height", vr.Height(),
				"round", vr.Round(),
			)
		} else {
			cs.Log().Debug(
				"acting member at this VoteResult; broadcast accept ballot",
				"vr", vr,
				"height", vr.Height(),
				"round", vr.Round(),
			)
			ballot, err = NewACCEPTBallot(
				cs.homeState.Home().Address(),
				cs.homeState.Block().Hash(),
				cs.homeState.Block().Round(),
				vr.Height(),
				block.Hash(),
				vr.Round(),
				vr.Proposal(),
			)
		}
	case StageACCEPT:
		ballot, err = NewINITBallot(
			cs.homeState.Home().Address(),
			cs.homeState.Block().Hash(),
			block.Hash(),
			block.Round(),
			block.Proposal(),
			block.Height().Add(1),
			Round(0),
		)
	}
	if err != nil {
		return err
	}

	if !ballot.Empty() {
		if err := ballot.Sign(cs.homeState.Home().PrivateKey(), nil); err != nil {
			return err
		}

		if err := cs.nt.Broadcast(ballot); err != nil {
			return err
		}
	}

	if err := cs.nextRoundTimer("wait-ballot-timeout-next-round-consensus", vr); err != nil {
		return err
	}

	return nil
}

func (cs *ConsensusStateHandler) prepareProposal(vr VoteResult) error {
	cs.Log().Debug("prepare proposal", "vr", vr)
	acting := cs.suffrage.Acting(vr.Height(), vr.Round())
	cs.Log().Debug("proposer selected", "acting", acting)
	if !acting.Proposer().Equal(cs.homeState.Home()) {
		cs.Log().Debug("proposer is not home; wait proposal")

		// NOTE wait proposal
		if err := cs.nextRoundTimer("proposal-timeout-next-round-consensus", vr); err != nil {
			return err
		}

		return nil
	}

	go func() {
		if err := cs.propose(vr); err != nil {
			cs.Log().Error("failed to propose", "error", err)
		}
	}()

	if err := cs.nextRoundTimer("proposal-timeout-next-round-consensus", vr); err != nil {
		return err
	}

	return nil
}

func (cs *ConsensusStateHandler) propose(vr VoteResult) error {
	cs.Log().Debug("proposer is home; propose new proposal")

	proposal, err := cs.proposalMaker.Make(vr.Height(), vr.Round(), cs.homeState.Block().Hash())
	if err != nil {
		return err
	}
	if err := proposal.Sign(cs.homeState.Home().PrivateKey(), nil); err != nil {
		return err
	}

	if err := cs.nt.Broadcast(proposal); err != nil {
		return err
	}

	return nil
}

func (cs *ConsensusStateHandler) startNextRound(vr VoteResult) error {
	ballot, err := NewINITBallot(
		cs.homeState.Home().Address(),
		cs.homeState.PreviousBlock().Hash(),
		cs.homeState.Block().Hash(),
		cs.homeState.Block().Round(),
		cs.homeState.Block().Proposal(),
		vr.Height(),
		vr.Round()+1,
	)
	if err != nil {
		return err
	}
	if err := ballot.Sign(cs.homeState.Home().PrivateKey(), nil); err != nil {
		return err
	}

	cs.Log().Debug("broadcast next round ballot", "vr", vr, "ballot", ballot)

	if err := cs.nt.Broadcast(ballot); err != nil {
		return err
	}

	return nil
}

func (cs *ConsensusStateHandler) nextRoundTimer(name string, vr VoteResult) error {
	if err := cs.stopTimer(); err != nil {
		return err
	}

	cs.Lock()
	defer cs.Unlock()

	cs.timer = common.NewCallbackTimer(
		name,
		cs.timeoutWaitBallot,
		func(common.Timer) error {
			return cs.startNextRound(vr)
		},
	)
	cs.timer.SetLogContext(nil, "node", cs.homeState.Home().Alias())
	if err := cs.timer.Start(); err != nil {
		return err
	}

	return nil
}
