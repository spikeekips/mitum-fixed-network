package contest_module

import (
	"encoding/json"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/contrib/contest/condition"
	contest_config "github.com/spikeekips/mitum/contrib/contest/config"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/isaac"
)

func init() {
	ProposalMakers = append(ProposalMakers, "ConditionProposalMaker")
	ProposalMakerConfigs["ConditionProposalMaker"] = ConditionProposalMakerConfig{}
}

type ConditionProposalMakerConfig struct {
	N          string                            `yaml:"name"`
	Conditions []*contest_config.ActionCondition `yaml:"conditions"`
	D          time.Duration                     `yaml:"delay,omitempty"`
}

func (cm ConditionProposalMakerConfig) Name() string {
	return cm.N
}

func (cm ConditionProposalMakerConfig) Delay() time.Duration {
	return cm.D
}

func (cm *ConditionProposalMakerConfig) IsValid() error {
	// NOTE empty condition be allowed.
	// if len(cm.Conditions) < 1 {
	// 	return xerrors.Errorf("empty `conditions`")
	// }

	for _, ca := range cm.Conditions {
		if err := ca.IsValid(); err != nil {
			return err
		}
	}

	return nil
}

func (cm *ConditionProposalMakerConfig) Merge(i interface{}) error {
	n, ok := interface{}(i).(ProposalMakerConfig)
	if !ok {
		return xerrors.Errorf("invalid merge source found: %%", i)
	}

	if cm.D < 1 {
		cm.D = n.Delay()
	}

	return nil
}

func (cm ConditionProposalMakerConfig) New(homeState *isaac.HomeState, l zerolog.Logger) isaac.ProposalMaker {
	var checkers []condition.ActionChecker
	for _, c := range cm.Conditions {
		checkers = append(checkers, c.ActionChecker())
	}

	cb := NewConditionProposalMaker(homeState, cm.Delay(), checkers)
	cb.SetLogger(l)

	return cb
}

type ConditionProposalMaker struct {
	*common.Logger
	isaac.DefaultProposalMaker
	homeState *isaac.HomeState
	checkers  []condition.ActionChecker
}

func NewConditionProposalMaker(homeState *isaac.HomeState, delay time.Duration, checkers []condition.ActionChecker) ConditionProposalMaker {
	return ConditionProposalMaker{
		DefaultProposalMaker: isaac.NewDefaultProposalMaker(homeState.Home(), delay),
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "condition-proposer_maker")
		}),
		homeState: homeState,
		checkers:  checkers,
	}
}

func (cp ConditionProposalMaker) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":     "ConditionProposalMaker",
		"checkers": cp.checkers,
	})
}

func (cp ConditionProposalMaker) Make(height isaac.Height, round isaac.Round, lastBlock hash.Hash) (isaac.Proposal, error) {
	if cp.checkers == nil {
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

	for _, c := range cp.checkers {
		if c.Checker().Check(li) {
			for _, action := range c.Actions() {
				cp.Log().Debug().
					Str("query", c.Checker().Query()).
					Str("action", action.Action()).
					RawJSON("data", li.Bytes()).
					Msg("condition matched")

				switch action.Action() {
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
	}

	return cp.DefaultProposalMaker.Make(height, round, lastBlock)
}
