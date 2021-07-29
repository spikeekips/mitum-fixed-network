package basicstates

import (
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
)

type JoiningState struct {
	*logging.Logging
	*BaseState
	local     *node.Local
	database  storage.Database
	policy    *isaac.LocalPolicy
	suffrage  base.Suffrage
	ballotbox *isaac.Ballotbox
}

func NewJoiningState(
	local *node.Local,
	st storage.Database,
	policy *isaac.LocalPolicy,
	suffrage base.Suffrage,
	ballotbox *isaac.Ballotbox,
) *JoiningState {
	return &JoiningState{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "basic-joining-state")
		}),
		BaseState: NewBaseState(base.StateJoining),
		local:     local,
		database:  st,
		policy:    policy,
		suffrage:  suffrage,
		ballotbox: ballotbox,
	}
}

func (st *JoiningState) Enter(sctx StateSwitchContext) (func() error, error) {
	callback := EmptySwitchFunc
	if i, err := st.BaseState.Enter(sctx); err != nil {
		return nil, err
	} else if i != nil {
		callback = i
	}

	voteproof := sctx.Voteproof()
	if voteproof == nil { // NOTE if empty voteproof, load last accept voteproof from database
		voteproof = st.database.LastVoteproof(base.StageACCEPT)
		if voteproof == nil {
			return nil, util.NotFoundError.Errorf("last accept voteproof not found")
		}
	}

	// NOTE prepare to broadcast INIT ballot
	return func() error {
		if err := callback(); err != nil {
			return err
		}

		// NOTE standalone node does not wait incoming ballots to join network
		if len(st.suffrage.Nodes()) < 2 {
			return st.broadcastINITBallotEnteredWithoutDelay(voteproof)
		}
		return st.broadcastINITBallotEntered(voteproof)
	}, nil
}

func (st *JoiningState) Exit(sctx StateSwitchContext) (func() error, error) {
	callback := EmptySwitchFunc
	if i, err := st.BaseState.Exit(sctx); err != nil {
		return nil, err
	} else if i != nil {
		callback = i
	}

	return func() error {
		if err := callback(); err != nil {
			return err
		}

		return st.Timers().StopTimers([]localtime.TimerID{TimerIDBroadcastJoingingINITBallot})
	}, nil
}

// ProcessVoteproof only receives the acceptable voteproof with last init voteproof
func (st *JoiningState) ProcessVoteproof(voteproof base.Voteproof) error {
	if voteproof.Stage() == base.StageINIT {
		if err := st.Timers().StopTimers([]localtime.TimerID{TimerIDBroadcastJoingingINITBallot}); err != nil {
			return err
		}

		return NewStateSwitchContext(base.StateJoining, base.StateConsensus).
			SetVoteproof(voteproof)
	}

	return nil
}

func (st *JoiningState) broadcastINITBallotEnteredWithoutDelay(voteproof base.Voteproof) error {
	var baseBallot ballot.INITV0
	if i, err := NextINITBallotFromACCEPTVoteproof(st.database, st.local, voteproof); err != nil {
		return err
	} else if err := i.Sign(st.local.Privatekey(), st.policy.NetworkID()); err != nil {
		return xerrors.Errorf("failed to re-sign joining INITBallot: %w", err)
	} else {
		baseBallot = i

		l := isaac.LoggerWithVoteproof(voteproof, st.Log())
		l.Debug().HintedVerbose("voteproof", voteproof, true).Msg("joining with latest accept voteproof from local")
	}

	timer := localtime.NewContextTimer(TimerIDBroadcastJoingingINITBallot, 0, func(i int) (bool, error) {
		if i%5 == 0 {
			_ = baseBallot.Sign(st.local.Privatekey(), st.policy.NetworkID())
		}

		if err := st.BroadcastBallot(baseBallot, i == 0); err != nil {
			st.Log().Error().Err(err).Msg("failed to broadcast init ballot")
		}

		return true, nil
	}).SetInterval(func(i int) time.Duration {
		if i < 1 {
			return time.Nanosecond
		}
		return st.policy.IntervalBroadcastingINITBallot()
	})

	if err := st.Timers().SetTimer(timer); err != nil {
		return err
	}

	return st.Timers().StartTimers([]localtime.TimerID{TimerIDBroadcastJoingingINITBallot}, true)
}

// broadcastINITBallotEntered broadcasts INIT ballot from local; it will be only
// executed when voteproof is stucked.
func (st *JoiningState) broadcastINITBallotEntered(voteproof base.Voteproof) error {
	var baseBallot ballot.INITV0
	if i, err := NextINITBallotFromACCEPTVoteproof(st.database, st.local, voteproof); err != nil {
		return err
	} else if err := i.Sign(st.local.Privatekey(), st.policy.NetworkID()); err != nil {
		return xerrors.Errorf("failed to re-sign joining INITBallot: %w", err)
	} else {
		baseBallot = i

		l := isaac.LoggerWithVoteproof(voteproof, st.Log())
		l.Debug().HintedVerbose("voteproof", voteproof, true).Msg("joining with latest accept voteproof from local")
	}

	checkBallotbox := st.checkBallotboxFunc()

	timer := localtime.NewContextTimer(TimerIDBroadcastJoingingINITBallot, 0, func(i int) (bool, error) {
		if err := checkBallotbox(); err != nil {
			return false, err
		}

		if i%5 == 0 {
			_ = baseBallot.Sign(st.local.Privatekey(), st.policy.NetworkID())
		}

		if err := st.BroadcastBallot(baseBallot, i == 0); err != nil {
			st.Log().Error().Err(err).Msg("failed to broadcast init ballot")
		}

		return true, nil
	}).SetInterval(func(i int) time.Duration {
		if i < 1 { // NOTE at first time, wait enough time for incoming ballot
			return st.policy.IntervalBroadcastingINITBallot() * 5
		}
		return st.policy.IntervalBroadcastingINITBallot()
	})

	if err := st.Timers().SetTimer(timer); err != nil {
		return err
	}

	return st.Timers().StartTimers([]localtime.TimerID{TimerIDBroadcastJoingingINITBallot}, true)
}

func (st *JoiningState) checkBallotboxFunc() func() error {
	var l sync.Mutex
	var last base.Voteproof
	return func() error {
		// NOTE find highest Ballot from ballotbox
		if i := st.ballotbox.LatestBallot(); i == nil {
			return nil
		} else if j, ok := i.(base.Voteproofer); ok {
			l.Lock()
			defer l.Unlock()

			vp := j.Voteproof()
			if base.CompareVoteproof(vp, st.LastVoteproof()) < 1 {
				return nil
			}

			if last == nil {
				last = vp
			} else if base.CompareVoteproof(vp, last) < 1 {
				return nil
			}

			go st.NewVoteproof(vp)
		}

		return nil
	}
}
