package contest_module

import (
	"encoding/json"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/contrib/contest/condition"
	contest_config "github.com/spikeekips/mitum/contrib/contest/config"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/isaac"
)

func init() {
	BallotMakers = append(BallotMakers, "ConditionBallotMaker")
	BallotMakerConfigs["ConditionBallotMaker"] = ConditionBallotMakerConfig{}
}

type ConditionBallotMakerConfig struct {
	N          string                            `yaml:"name"`
	Conditions []*contest_config.ActionCondition `yaml:"conditions"`
}

func (cb ConditionBallotMakerConfig) Name() string {
	return cb.N
}

func (cb *ConditionBallotMakerConfig) IsValid() error {
	// NOTE empty condition be allowed.
	// if len(cb.Conditions) < 1 {
	// 	return xerrors.Errorf("empty `conditions`")
	// }

	for _, ca := range cb.Conditions {
		if err := ca.IsValid(); err != nil {
			return err
		}
	}

	return nil
}

func (cb *ConditionBallotMakerConfig) Merge(interface{}) error {
	return nil
}

func (cb ConditionBallotMakerConfig) New(homeState *isaac.HomeState, l zerolog.Logger) isaac.BallotMaker {
	var checkers []condition.ActionChecker
	for _, c := range cb.Conditions {
		checkers = append(checkers, c.ActionChecker())
	}

	bm := NewConditionBallotMaker(homeState, checkers)
	bm.SetLogger(l)

	return bm
}

type ConditionBallotMaker struct {
	*common.Logger
	isaac.DefaultBallotMaker
	homeState *isaac.HomeState
	checkers  []condition.ActionChecker
}

func NewConditionBallotMaker(homeState *isaac.HomeState, checkers []condition.ActionChecker) ConditionBallotMaker {
	return ConditionBallotMaker{
		DefaultBallotMaker: isaac.NewDefaultBallotMaker(homeState.Home()),
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "condition-ballot_maker")
		}),
		homeState: homeState,
		checkers:  checkers,
	}
}

func (cb ConditionBallotMaker) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":     "ConditionBallotMaker",
		"checkers": cb.checkers,
	})
}

func (cb ConditionBallotMaker) modifyBallot(
	lastBlock hash.Hash,
	lastRound isaac.Round,
	nextHeight isaac.Height,
	nextBlock hash.Hash,
	currentRound isaac.Round,
	currentProposal hash.Hash,
	stage isaac.Stage,
) (isaac.Ballot, error) {
	if cb.checkers != nil {
		li, err := condition.NewLogItemFromMap(
			map[string]interface{}{
				"node":  cb.homeState.Home().Alias(),
				"state": cb.homeState.State().String(),
				"block": map[string]interface{}{
					"height":   cb.homeState.Block().Height().Uint64(),
					"round":    cb.homeState.Block().Round().Uint64(),
					"proposal": cb.homeState.Block().Proposal().String(),
				},
				"previousBlock": map[string]interface{}{
					"height":   cb.homeState.PreviousBlock().Height().Uint64(),
					"round":    cb.homeState.PreviousBlock().Round().Uint64(),
					"proposal": cb.homeState.PreviousBlock().Proposal().String(),
				},
				"ballot": map[string]interface{}{
					"stage":            stage.String(),
					"last_block":       lastBlock.String(),
					"last_round":       lastRound.Uint64(),
					"next_height":      nextHeight.Uint64(),
					"next_block":       nextBlock.String(),
					"current_round":    currentRound.Uint64(),
					"current_proposal": currentProposal.String(),
				},
			})
		if err != nil {
			return isaac.Ballot{}, err
		}

		for _, c := range cb.checkers {
			if c.Checker().Check(li) {
				for _, action := range c.Actions() {
					cb.Log().Debug().
						Str("query", c.Checker().Query()).
						Str("action", action.Action()).
						RawJSON("data", li.Bytes()).
						Msg("condition matched")
					switch action.Action() {
					case "empty-ballot":
						return isaac.Ballot{}, xerrors.Errorf("empty ballot by force")
					case "random-last_block":
						lastBlock = NewRandomBlockHash()
					case "random-last_round":
						lastRound = NewRandomRound()
					case "random-next_height":
						nextHeight = NewRandomHeight()
					case "random-next_block":
						nextBlock = NewRandomBlockHash()
					case "random-current_round":
						currentRound = NewRandomRound()
					case "random-current_proposal":
						currentProposal = NewRandomProposalHash()
					}
				}
			}
		}
	}

	var cbFunc func(hash.Hash, isaac.Round, isaac.Height, hash.Hash, isaac.Round, hash.Hash) (isaac.Ballot, error)

	switch stage {
	case isaac.StageINIT:
		cbFunc = cb.DefaultBallotMaker.INIT
	case isaac.StageSIGN:
		cbFunc = cb.DefaultBallotMaker.SIGN
	case isaac.StageACCEPT:
		cbFunc = cb.DefaultBallotMaker.ACCEPT
	default:
		err := xerrors.Errorf("unknown stage found")
		cb.Log().Error().
			Err(err).
			Str("stage", stage.String()).
			Send()
		return isaac.Ballot{}, err
	}

	return cbFunc(
		lastBlock, lastRound, nextHeight, nextBlock, currentRound, currentProposal,
	)
}

func (cb ConditionBallotMaker) INIT(
	lastBlock hash.Hash,
	lastRound isaac.Round,
	nextHeight isaac.Height,
	nextBlock hash.Hash,
	currentRound isaac.Round,
	currentProposal hash.Hash,
) (isaac.Ballot, error) {
	return cb.modifyBallot(
		lastBlock, lastRound, nextHeight, nextBlock, currentRound, currentProposal,
		isaac.StageINIT,
	)
}

func (cb ConditionBallotMaker) SIGN(
	lastBlock hash.Hash,
	lastRound isaac.Round,
	nextHeight isaac.Height,
	nextBlock hash.Hash,
	currentRound isaac.Round,
	currentProposal hash.Hash,
) (isaac.Ballot, error) {
	return cb.modifyBallot(
		lastBlock, lastRound, nextHeight, nextBlock, currentRound, currentProposal,
		isaac.StageSIGN,
	)
}

func (cb ConditionBallotMaker) ACCEPT(
	lastBlock hash.Hash,
	lastRound isaac.Round,
	nextHeight isaac.Height,
	nextBlock hash.Hash,
	currentRound isaac.Round,
	currentProposal hash.Hash,
) (isaac.Ballot, error) {
	return cb.modifyBallot(
		lastBlock, lastRound, nextHeight, nextBlock, currentRound, currentProposal,
		isaac.StageACCEPT,
	)
}
