package contest_module

import (
	"encoding/json"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/contrib/contest/condition"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
	"golang.org/x/xerrors"
)

type ConditionBallotHandler struct {
	checker condition.ConditionChecker
	action  string
}

func NewConditionBallotHandler(checker condition.ConditionChecker, action string) ConditionBallotHandler {
	return ConditionBallotHandler{checker: checker, action: action}
}

type ConditionBallotMaker struct {
	*common.Logger
	isaac.DefaultBallotMaker
	home       node.Home
	conditions map[string]ConditionBallotHandler
}

func NewConditionBallotMaker(home node.Home, conditions map[string]ConditionBallotHandler) ConditionBallotMaker {
	return ConditionBallotMaker{
		DefaultBallotMaker: isaac.NewDefaultBallotMaker(home),
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "condition-ballot_maker")
		}),
		home:       home,
		conditions: conditions,
	}
}

func (cb ConditionBallotMaker) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":       "ConditionBallotMaker",
		"conditions": cb.conditions,
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
	if cb.conditions != nil {
		li, err := condition.NewLogItemFromMap(
			map[string]interface{}{
				"node":             cb.home.Alias(),
				"stage":            stage.String(),
				"last_block":       lastBlock.String(),
				"last_round":       lastRound,
				"next_height":      nextHeight.Uint64(),
				"next_block":       nextBlock,
				"current_round":    currentRound,
				"current_proposal": currentProposal,
			})
		if err != nil {
			return isaac.Ballot{}, err
		}

		for name, c := range cb.conditions {
			if c.checker.Check(li) {
				cb.Log().Debug().
					Str("checker", name).
					Str("query", c.checker.Query()).
					Str("action", c.action).
					Interface("item", li).
					Msg("condition matched")
				switch c.action {
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
