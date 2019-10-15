package isaac

import (
	"context"
	"sync"

	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/common"
)

type Compiler struct {
	sync.RWMutex
	*common.Logger
	homeState            *HomeState
	ballotbox            *Ballotbox
	lastINITVoteResult   VoteResult
	lastStagesVoteResult VoteResult
	ballotChecker        *common.ChainChecker
}

func NewCompiler(homeState *HomeState, ballotbox *Ballotbox, ballotChecker *common.ChainChecker) *Compiler {
	return &Compiler{
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "compiler")
		}),
		homeState:     homeState,
		ballotbox:     ballotbox,
		ballotChecker: ballotChecker,
	}
}

func (cm *Compiler) SetLogger(l zerolog.Logger) *common.Logger {
	_ = cm.Logger.SetLogger(l)
	_ = cm.ballotChecker.SetLogger(l)
	_ = cm.ballotbox.SetLogger(l)

	return cm.Logger
}

func (cm *Compiler) AddLoggerContext(cf func(c zerolog.Context) zerolog.Context) *common.Logger {
	_ = cm.Logger.AddLoggerContext(cf)
	_ = cm.ballotChecker.AddLoggerContext(cf)

	return cm.Logger
}

func (cm *Compiler) Vote(ballot Ballot) (VoteResult, error) {
	err := cm.ballotChecker.
		New(context.TODO()).
		SetContext("ballot", ballot).
		SetContext("lastINITVoteResult", cm.LastINITVoteResult()).
		SetContext("lastStagesVoteResult", cm.LastStagesVoteResult()).
		Check()
	if err != nil {
		return VoteResult{}, err
	}

	cm.Log().Debug().Object("ballot", ballot.Hash()).Msg("ballot checked")

	vr, err := cm.ballotbox.Vote(
		ballot.Node(),
		ballot.Height(),
		ballot.Round(),
		ballot.Stage(),
		ballot.Block(),
		ballot.LastBlock(),
		ballot.LastRound(),
		ballot.Proposal(),
	)
	if err != nil {
		return VoteResult{}, err
	}

	if vr.IsClosed() || !vr.IsFinished() {
		return VoteResult{}, nil
	} else if vr.GotMajority() {
		switch vr.Stage() {
		case StageINIT:
			cm.SetLastINITVoteResult(vr)

			// NOTE remove vote records,
			// - other heights
			// - same height, but lower round
			cm.ballotbox.Tidy(vr.Height(), vr.Round())
		default:
			cm.SetLastStagesVoteResult(vr)
		}
	}

	return vr, nil
}

func (cm *Compiler) LastINITVoteResult() VoteResult {
	cm.RLock()
	defer cm.RUnlock()

	return cm.lastINITVoteResult
}

func (cm *Compiler) SetLastINITVoteResult(vr VoteResult) {
	cm.Lock()
	defer cm.Unlock()

	cm.Log().Debug().
		Object("previous_vr", cm.lastINITVoteResult).
		Object("new_vr", vr).
		Msg("set last init vote result")

	cm.lastINITVoteResult = vr
}

func (cm *Compiler) LastStagesVoteResult() VoteResult {
	cm.RLock()
	defer cm.RUnlock()

	return cm.lastStagesVoteResult
}

func (cm *Compiler) SetLastStagesVoteResult(vr VoteResult) {
	cm.Lock()
	defer cm.Unlock()

	cm.lastStagesVoteResult = vr
}
