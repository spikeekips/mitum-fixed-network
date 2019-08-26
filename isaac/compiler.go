package isaac

import (
	"context"
	"sync"

	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/common"
)

type Compiler struct {
	sync.RWMutex
	*common.ZLogger
	homeState            *HomeState
	ballotbox            *Ballotbox
	lastINITVoteResult   VoteResult
	lastStagesVoteResult VoteResult
	ballotChecker        *common.ChainChecker
}

func NewCompiler(homeState *HomeState, ballotbox *Ballotbox, ballotChecker *common.ChainChecker) *Compiler {
	return &Compiler{
		ZLogger: common.NewZLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "compiler")
		}),
		homeState:     homeState,
		ballotbox:     ballotbox,
		ballotChecker: ballotChecker,
	}
}

func (cm *Compiler) SetLogger(l zerolog.Logger) *common.ZLogger {
	_ = cm.ZLogger.SetLogger(l)
	_ = cm.ballotChecker.SetLogger(l)

	return cm.ZLogger
}

func (cm *Compiler) AddLoggerContext(cf func(c zerolog.Context) zerolog.Context) *common.ZLogger {
	_ = cm.ZLogger.AddLoggerContext(cf)
	_ = cm.ballotChecker.AddLoggerContext(cf)

	return cm.ZLogger
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

	cm.Log().Debug().Interface("ballot", ballot).Msg("ballot checked")

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
	}

	switch vr.Stage() {
	case StageINIT:
		cm.SetLastINITVoteResult(vr)
	default:
		cm.SetLastStagesVoteResult(vr)
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
