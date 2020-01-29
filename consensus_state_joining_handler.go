package mitum

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
)

/*
ConsensusStateJoiningHandler tries to join network safely. This is the basic
strategy,

* Keeping broadcasting INIT ballot with VoteResult

- waits the incoming INIT ballots, which have VoteResult.
- if timed out, still broadcasts and wait.

* With VoteResult

- if height of VoteResult is not the next of local block
	-> moves to sync

- if next of local block,
	- if INIT VR,
		-> moves to consensus

	- if ACCEPT VR,
		1. processes Proposal.
		1. check the result of new block of Proposal.
		1. if not,
			-> moves to sync.
		1. waits next INIT VR

		- processing may be late before next INIT VR.
		- with next INIT VR and next Proposal, waits next next INIT VR.
*/
type ConsensusStateJoiningHandler struct {
	*logging.Logger
	broadcastingINITBallotTimer util.Daemon
	localState                  *LocalState
}

func NewConsensusStateJoiningHandler(
	localState *LocalState,
) (*ConsensusStateJoiningHandler, error) {
	cs := &ConsensusStateJoiningHandler{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "consensus-state-joining-handler")
		}),
		localState: localState,
	}

	bt, err := localtime.NewCallbackTimer(
		"joining-broadcasting-init-ballot",
		cs.broadcastingINITBallot,
		0,
		func() time.Duration {
			return localState.Policy().IntervalBroadcastingINITBallotInJoining()
		},
	)
	if err != nil {
		return nil, err
	}
	cs.broadcastingINITBallotTimer = bt

	return cs, nil
}

func (cs ConsensusStateJoiningHandler) State() ConsensusState {
	return ConsensusStateJoining
}

func (cs *ConsensusStateJoiningHandler) Activate() error {
	// starts to keep broadcasting INIT Ballot
	if err := cs.startbroadcastingINITBallotTimer(); err != nil {
		return err
	}

	return nil
}

func (cs *ConsensusStateJoiningHandler) Deactivate() error {
	if err := cs.stopbroadcastingINITBallotTimer(); err != nil {
		return err
	}
	return nil
}

func (cs *ConsensusStateJoiningHandler) startbroadcastingINITBallotTimer() error {
	if err := cs.broadcastingINITBallotTimer.Stop(); err != nil {
		if !xerrors.Is(err, util.DaemonAlreadyStoppedError) {
			return err
		}
	}

	return cs.broadcastingINITBallotTimer.Start()
}

func (cs *ConsensusStateJoiningHandler) stopbroadcastingINITBallotTimer() error {
	if err := cs.broadcastingINITBallotTimer.Stop(); err != nil && !xerrors.Is(err, util.DaemonAlreadyStoppedError) {
		return err
	}

	return nil
}

func (cs *ConsensusStateJoiningHandler) broadcastingINITBallot() (bool, error) {
	ib := INITBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: cs.localState.Node().Address(),
		},
		INITBallotV0Fact: INITBallotV0Fact{
			BaseBallotV0Fact: BaseBallotV0Fact{
				height: cs.localState.LastBlockHeight() + 1,
				round:  Round(0),
			},
			previousBlock: cs.localState.LastBlockHash(),
			previousRound: cs.localState.LastBlockRound(),
		},
	}

	// TODO NetworkID must be given.
	if err := ib.Sign(cs.localState.Node().Privatekey(), nil); err != nil {
		cs.Log().Error().Err(err).Msg("failed to broadcast INIT ballot; will keep trying")
		return true, nil
	}

	cs.localState.Nodes().Traverse(func(n Node) bool {
		go func(n Node) {
			if err := n.Channel().SendSeal(ib); err != nil {
				cs.Log().Error().Err(err).Msg("failed to broadcast INIT ballot; will keep trying")
			}
		}(n)

		return true
	})

	return true, nil
}

// NewSeal only cares on INIT ballot and it's VoteResult.
func (cs *ConsensusStateJoiningHandler) NewSeal(sl seal.Seal) error {
	var ballot INITBallot
	switch t := sl.(type) {
	case INITBallot:
		ballot = t
	default:
		return nil
	}

	fmt.Println(">", ballot)

	return nil
}

func (cs *ConsensusStateJoiningHandler) NewVoteResult(vr *VoteResult) error {
	if err := cs.stopbroadcastingINITBallotTimer(); err != nil {
		return err
	}

	fmt.Println(">", vr)

	return nil
}
