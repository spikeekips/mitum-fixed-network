package basicstates

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
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
	local     node.Local
	database  storage.Database
	policy    *isaac.LocalPolicy
	suffrage  base.Suffrage
	ballotbox *isaac.Ballotbox
}

func NewJoiningState(
	local node.Local,
	db storage.Database,
	policy *isaac.LocalPolicy,
	suffrage base.Suffrage,
	ballotbox *isaac.Ballotbox,
) *JoiningState {
	return &JoiningState{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "basic-joining-state")
		}),
		BaseState: NewBaseState(base.StateJoining),
		local:     local,
		database:  db,
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

		// NOTE standalone node does not wait incoming ballots to join consensus
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

		return st.NewStateSwitchContext(base.StateConsensus).
			SetVoteproof(voteproof)
	}

	return nil
}

func (st *JoiningState) broadcastINITBallotEnteredWithoutDelay(voteproof base.Voteproof) error {
	if st.underHandover() {
		return nil
	}

	baseBallot, err := NextINITBallotFromACCEPTVoteproof(st.database, st.local, voteproof, st.policy.NetworkID())
	if err != nil {
		return err
	}

	l := st.Log().With().Str("voteproof_id", voteproof.ID()).Logger()
	l.Trace().Interface("voteproof", voteproof).Msg("joining with latest accept voteproof from local")
	l.Debug().Object("voteproof", voteproof).Msg("joining with latest accept voteproof from local")

	timer := localtime.NewContextTimer(TimerIDBroadcastJoingingINITBallot, 0, func(i int) (bool, error) {
		if i%5 == 0 {
			_ = signBallotWithFact(
				baseBallot,
				st.local.Address(),
				st.local.Privatekey(),
				st.policy.NetworkID(),
			)
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
	if st.underHandover() {
		return nil
	}

	baseBallot, err := NextINITBallotFromACCEPTVoteproof(st.database, st.local, voteproof, st.policy.NetworkID())
	if err != nil {
		return err
	}

	l := st.Log().With().Str("voteproof_id", voteproof.ID()).Logger()
	l.Trace().Interface("voteproof", voteproof).Msg("joining with latest accept voteproof from local")
	l.Debug().Object("voteproof", voteproof).Msg("joining with latest accept voteproof from local")

	checkBallotbox := st.checkBallotboxFunc()

	timer := localtime.NewContextTimer(TimerIDBroadcastJoingingINITBallot, 0, func(i int) (bool, error) {
		if err := checkBallotbox(); err != nil {
			return false, err
		}

		if i%5 == 0 {
			_ = signBallotWithFact(
				baseBallot,
				st.local.Address(),
				st.local.Privatekey(),
				st.policy.NetworkID(),
			)
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
		i := st.ballotbox.LatestBallot()
		if i == nil {
			return nil
		}

		l.Lock()
		defer l.Unlock()

		vp := i.BaseVoteproof()
		if base.CompareVoteproof(vp, st.LastVoteproof()) < 1 {
			return nil
		}

		if last == nil {
			last = vp
		} else if base.CompareVoteproof(vp, last) < 1 {
			return nil
		}

		go st.NewVoteproof(vp)

		return nil
	}
}
