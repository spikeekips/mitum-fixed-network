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

// ConsensusStateHandler is only for suffrage node
type ConsensusStateHandler struct {
	sync.RWMutex
	*common.Logger
	homeState         *HomeState
	compiler          *Compiler
	nt                network.Network
	suffrage          Suffrage
	ballotMaker       BallotMaker
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
	ballotMaker BallotMaker,
	proposalValidator ProposalValidator,
	proposalMaker ProposalMaker,
	timeoutWaitBallot time.Duration,
) (*ConsensusStateHandler, error) {
	if homeState.PreviousBlock().Empty() {
		return nil, xerrors.Errorf("previous block is empty")
	}

	return &ConsensusStateHandler{
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "s.h.consensus")
		}),
		homeState:         homeState,
		compiler:          compiler,
		nt:                nt,
		suffrage:          suffrage,
		ballotMaker:       ballotMaker,
		proposalValidator: proposalValidator,
		proposalMaker:     proposalMaker,
		timeoutWaitBallot: timeoutWaitBallot,
		proposalChecker:   NewProposalCheckerConsensus(homeState, suffrage),
		voteResultChecker: NewConsensusVoteResultChecker(homeState),
	}, nil
}

func (cs *ConsensusStateHandler) Start() error {
	if cs.timeoutWaitBallot < time.Nanosecond {
		cs.Log().Warn().Dur("timeout", cs.timeoutWaitBallot).Msg("timeoutWaitBallot is too short")
	}

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
			cs.Log().Error().Err(err).Msg("failed to propose")
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
		cs.Log().Error().Err(err).Msg("failed to stop timer")
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

	cs.Log().Debug().Object("proposal", proposal.Hash()).Msg("proposal checked")

	err = cs.nextRoundTimer(
		"ballot-timeout",
		cs.compiler.LastINITVoteResult(),
	)
	if err != nil {
		return err
	}

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
		cs.Log().Debug().
			Object("proposal", proposal.Hash()).
			Uint64("height", proposal.Height().Uint64()).
			Uint64("round", proposal.Round().Uint64()).
			Object("acting", acting).
			Msg("not acting member at this proposal; not broadcast sign ballot")
	} else {
		cs.Log().Debug().
			Object("proposal", proposal.Hash()).
			Uint64("height", proposal.Height().Uint64()).
			Uint64("round", proposal.Round().Uint64()).
			Object("acting", acting).
			Msg("acting member at this proposal; broadcast sign ballot")
	}

	if insideActing {
		ballot, err := cs.ballotMaker.SIGN(
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

	cs.Log().Debug().Object("vr", vr).Msg("VoteResult checked")

	if vr.GotDraw() {
		cs.Log().Debug().Object("vr", vr).Msg("VoteResult drawed")
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
		cs.Log().Debug().Object("vr", vr).Msg("got VoteResult of next block; keep going")

		if !cs.homeState.Block().Hash().Equal(vr.LastBlock()) {
			cs.Log().Error().
				Object("home", cs.homeState.Block().Hash()).
				Object("block_vr", vr.LastBlock()).
				Object("vr", vr).
				Msg("init for next block; last block does not match; move to sync")
			cs.chanState <- NewStateContext(node.StateSyncing).
				SetContext("vr", vr)

			return xerrors.Errorf("init for next block; last block does not match; move to sync")
		}

		block, err := cs.proposalValidator.NewBlock(vr.Height(), vr.Round(), vr.Proposal())
		if err != nil {
			cs.Log().Error().
				Err(err).
				Object("vr", vr).
				Msg("failed to make new block from VoteResult")
			return err
		}

		_ = cs.homeState.SetBlock(block)

		cs.Log().Info().Object("block", block).Object("vr", vr).Msg("new block created")
	case diff == 1: // next round
		cs.Log().Debug().Object("vr", vr).Msg("got VoteResult of next round; keep going")

		if !cs.homeState.Block().Hash().Equal(vr.Block()) {
			cs.Log().Error().
				Object("home", cs.homeState.Block().Hash()).
				Object("block_vr", vr.Block()).
				Object("vr", vr).
				Msg("init for next round; block does not match; move to sync")
			cs.chanState <- NewStateContext(node.StateSyncing).
				SetContext("vr", vr)

			return xerrors.Errorf("init for next round; block does not match; move to sync")
		}
	default: // unexpected height received, move to sync
		cs.Log().Debug().Object("vr", vr).Msg("got not expected height VoteResult; move to sync")
		cs.chanState <- NewStateContext(node.StateSyncing).
			SetContext("vr", vr)
		return xerrors.Errorf("got not expected height VoteResult; move to sync")
	}

	return cs.prepareProposal(vr)
}

func (cs *ConsensusStateHandler) gotNotINITMajority(vr VoteResult) error {
	switch vr.Stage() {
	case StageSIGN, StageACCEPT:
	default:
		return xerrors.Errorf("invalid stage found", "vr", vr)
	}

	if !cs.proposalValidator.Validated(vr.Proposal()) {
		cs.Log().Debug().Object("vr", vr).Msg("proposal did not validated; validate it")
	}

	block, err := cs.proposalValidator.NewBlock(vr.Height(), vr.Round(), vr.Proposal())
	if err != nil {
		cs.Log().Error().Err(err).Object("vr", vr).Msg("failed to make new block from proposal")
		return err
	}

	if !vr.Block().Equal(block.Hash()) {
		cs.Log().Warn().
			Object("vr_block", vr.Block()).
			Object("block", block.Hash()).
			Object("vr", vr).
			Msg("block hash does not match")
	}

	var ballot Ballot
	switch vr.Stage() {
	case StageSIGN:
		acting := cs.suffrage.Acting(vr.Height(), vr.Round())
		if !acting.Exists(cs.homeState.Home().Address()) {
			cs.Log().Debug().
				Object("vr", vr).
				Uint64("height", vr.Height().Uint64()).
				Uint64("round", vr.Round().Uint64()).
				Object("acting", acting).
				Msg("not acting member at this VoteResult; not broadcast accept ballot")
		} else {
			cs.Log().Debug().
				Object("vr", vr).
				Uint64("height", vr.Height().Uint64()).
				Uint64("round", vr.Round().Uint64()).
				Object("acting", acting).
				Msg("acting member at this VoteResult; broadcast accept ballot")
			ballot, err = cs.ballotMaker.ACCEPT(
				cs.homeState.Block().Hash(),
				cs.homeState.Block().Round(),
				vr.Height(),
				block.Hash(),
				vr.Round(),
				vr.Proposal(),
			)
		}
	case StageACCEPT:
		ballot, err = cs.ballotMaker.INIT(
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
		if err := cs.nt.Broadcast(ballot); err != nil {
			return err
		}
	}

	if err := cs.nextRoundTimer("ballot-timeout", vr); err != nil {
		return err
	}

	return nil
}

func (cs *ConsensusStateHandler) prepareProposal(vr VoteResult) error {
	cs.Log().Debug().Object("vr", vr).Msg("prepare proposal")
	acting := cs.suffrage.Acting(vr.Height(), vr.Round())
	cs.Log().Debug().Object("acting", acting).Msg("proposer selected")
	if !acting.Proposer().Equal(cs.homeState.Home()) {
		cs.Log().Debug().Msg("proposer is not home; wait proposal")

		// NOTE wait proposal
		if err := cs.nextRoundTimer("proposal-timeout", vr); err != nil {
			return err
		}

		return nil
	}

	if err := cs.propose(vr); err != nil {
		cs.Log().Error().Err(err).Msg("failed to propose")
	}

	if err := cs.nextRoundTimer("proposal-timeout", vr); err != nil {
		return err
	}

	return nil
}

func (cs *ConsensusStateHandler) propose(vr VoteResult) error {
	cs.Log().Debug().Object("vr", vr).Msg("proposer is home; propose new proposal")

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
	ballot, err := cs.ballotMaker.INIT(
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

	cs.Log().Debug().Object("vr", vr).Object("ballot", ballot).Msg("broadcast next round ballot")

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
	cs.timer.SetLogger(*cs.Log())
	if err := cs.timer.Start(); err != nil {
		return err
	}

	return nil
}
