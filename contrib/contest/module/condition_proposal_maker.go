package contest_module

import (
	"encoding/json"
	"time"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/contrib/contest/condition"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/isaac"
	"golang.org/x/xerrors"
)

type ConditionProposalMaker struct {
	*common.Logger
	isaac.DefaultProposalMaker
	homeState  *isaac.HomeState
	conditions map[string]condition.Action
}

func NewConditionProposalMaker(homeState *isaac.HomeState, delay time.Duration, conditions map[string]condition.Action) ConditionProposalMaker {
	return ConditionProposalMaker{
		DefaultProposalMaker: isaac.NewDefaultProposalMaker(homeState.Home(), delay),
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "condition-proposer_maker")
		}),
		homeState:  homeState,
		conditions: conditions,
	}
}

func (cp ConditionProposalMaker) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":       "ConditionProposalMaker",
		"conditions": cp.conditions,
	})
}

func (cp ConditionProposalMaker) Make(height isaac.Height, round isaac.Round, lastBlock hash.Hash) (isaac.Proposal, error) {
	if cp.conditions == nil {
		return cp.DefaultProposalMaker.Make(height, round, lastBlock)
	}

	li, err := condition.NewLogItemFromMap(
		map[string]interface{}{
			"node":  cp.homeState.Home().Alias(),
			"state": cp.homeState.State().String(),
			"block": map[string]interface{}{
				"height":   cp.homeState.Block().Height().Uint64(),
				"round":    cp.homeState.Block().Round().Uint64(),
				"proposal": cp.homeState.Block().Proposal().String(),
			},
			"previousBlock": map[string]interface{}{
				"height":   cp.homeState.PreviousBlock().Height().Uint64(),
				"round":    cp.homeState.PreviousBlock().Round().Uint64(),
				"proposal": cp.homeState.PreviousBlock().Proposal().String(),
			},
			"proposal": map[string]interface{}{
				"height":     height.Uint64(),
				"round":      round.Uint64(),
				"last_block": lastBlock.String(),
			},
		})
	if err != nil {
		return isaac.Proposal{}, err
	}

	for name, c := range cp.conditions {
		if c.Checker().Check(li) {
			cp.Log().Debug().
				Str("checker", name).
				Str("query", c.Checker().Query()).
				Str("action", c.Action()).
				RawJSON("data", li.Bytes()).
				Msg("condition matched")

			switch c.Action() {
			case "empty-proposal":
				return isaac.Proposal{}, xerrors.Errorf("empty proposal by force")
			case "random-last_block":
				lastBlock = NewRandomBlockHash()
			case "random-round":
				var r isaac.Round
				for {
					r = NewRandomRound()
					if r == round {
						continue
					}
					round = r
					break
				}
			case "random-height":
				var h isaac.Height
				for {
					h = NewRandomHeight()
					if height.Equal(h) {
						continue
					}
					height = h
					break
				}
			}
		}
	}

	return cp.DefaultProposalMaker.Make(height, round, lastBlock)
}
