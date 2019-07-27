package isaac

import (
	"sync"

	"github.com/inconshreveable/log15"
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

func NewCompiler(homeState *HomeState, ballotbox *Ballotbox) *Compiler {
	return &Compiler{
		Logger:        common.NewLogger(log, "module", "compiler"),
		homeState:     homeState,
		ballotbox:     ballotbox,
		ballotChecker: NewCompilerBallotChecker(homeState),
	}
}

func (cm *Compiler) SetLogContext(ctx log15.Ctx, args ...interface{}) *common.Logger {
	_ = cm.Logger.SetLogContext(ctx, args...)
	_ = cm.ballotChecker.SetLogContext(ctx, args...)

	return cm.Logger
}

func (cm *Compiler) Vote(ballot Ballot) (VoteResult, error) {
	cm.Lock()
	defer cm.Unlock()

	err := cm.ballotChecker.
		New(nil).
		SetContext("ballot", ballot).
		SetContext("lastINITVoteResult", cm.lastINITVoteResult).
		SetContext("lastStagesVoteResult", cm.lastStagesVoteResult).
		Check()
	if err != nil {
		return VoteResult{}, err
	}

	vr, err := cm.ballotbox.Vote(
		ballot.Node(),
		ballot.Height(),
		ballot.Round(),
		ballot.Stage(),
		ballot.Block(),
		ballot.LastBlock(),
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
		cm.lastINITVoteResult = vr
	default:
		cm.lastStagesVoteResult = vr
	}

	return vr, nil
}

func (cm *Compiler) LastINITVoteResult() VoteResult {
	cm.RLock()
	defer cm.RUnlock()

	return cm.lastINITVoteResult
}
