package mitum

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/util"
)

/*
ConsensusStateJoiningHandler tries to join network safely. This is the basic
strategy,

* Keeping Broadcasting INIT ballot
	- waits the new VR(VoteResult)
	- if timed out, requests VP(VoteProof)

* With VoteResult

- if height of VoteResult is not the next of local block
	-> moves to sync

- if not,
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

* Requesting VoteProof(INIT)

1. requests VP(VoteProof) to the suffrage members.
1. if requesting VP is timeed out
	-> keeps requesting.

* With VoteProof

	VP has,
		- height
		- round
		- previous block hash
		- previous round
		- VoteRecord(s) of suffrage members

- if height of VP is not the next of local block
	-> moves to syncing state.

- if not,
	-> keeps broadcasting INIT ballot by round of VP
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
	// TODO set valid values in INIT ballot
	ib := INITBallotV0{
		BaseBallotV0: BaseBallotV0{
			height: cs.localState.LastBlockHeight() + 1,
			round:  Round(0),
			node:   cs.localState.Node().Address(),
		},
		previousBlock: cs.localState.LastBlockHash(),
		previousRound: cs.localState.LastBlockRound(),
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

func (cs *ConsensusStateJoiningHandler) NewProposal(pr Proposal) error {
	fmt.Println(">", pr)
	return nil
}

func (cs *ConsensusStateJoiningHandler) NewVoteResult(vr VoteResult) error {
	if err := cs.stopbroadcastingINITBallotTimer(); err != nil {
		return err
	}

	fmt.Println(">", vr)
	return nil
}
