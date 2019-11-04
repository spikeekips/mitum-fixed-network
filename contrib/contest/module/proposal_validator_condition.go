package contest_module

import (
	"encoding/json"
	"reflect"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/contrib/contest/condition"
	contest_config "github.com/spikeekips/mitum/contrib/contest/config"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/isaac"
)

func init() {
	ProposalValidators = append(ProposalValidators, "ConditionProposalValidator")
	ProposalValidatorConfigs["ConditionProposalValidator"] = ConditionProposalValidatorConfig{}
}

type ConditionProposalValidatorConfig struct {
	N          string                            `yaml:"name"`
	Conditions []*contest_config.ActionCondition `yaml:"conditions"`
}

func (cm ConditionProposalValidatorConfig) Name() string {
	return cm.N
}

func (cm *ConditionProposalValidatorConfig) IsValid() error {
	for _, ca := range cm.Conditions {
		if err := ca.IsValid(); err != nil {
			return err
		}
	}

	return nil
}

func (cm *ConditionProposalValidatorConfig) Merge(i interface{}) error {
	if _, ok := interface{}(i).(ProposalValidatorConfig); !ok {
		return xerrors.Errorf("invalid merge source found: %%", i)
	}

	return nil
}

func (cm ConditionProposalValidatorConfig) New(homeState *isaac.HomeState, l zerolog.Logger) isaac.ProposalValidator {
	var checkers []condition.ActionChecker
	for _, c := range cm.Conditions {
		checkers = append(checkers, c.ActionChecker())
	}

	cb := NewConditionProposalValidator(homeState, checkers)
	cb.SetLogger(l)

	return cb
}

type ConditionProposalValidator struct {
	*common.Logger
	DefaultProposalValidator
	homeState *isaac.HomeState
	checkers  []condition.ActionChecker
}

func NewConditionProposalValidator(homeState *isaac.HomeState, checkers []condition.ActionChecker) ConditionProposalValidator {
	return ConditionProposalValidator{
		DefaultProposalValidator: NewDefaultProposalValidator(homeState),
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "condition-proposer_validator")
		}),
		homeState: homeState,
		checkers:  checkers,
	}
}

func (dp ConditionProposalValidator) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":     "ConditionProposalValidator",
		"checkers": dp.checkers,
	})
}

func (dp ConditionProposalValidator) Validated(proposal hash.Hash) bool {
	return dp.DefaultProposalValidator.Validated(proposal)
}

func (dp ConditionProposalValidator) NewBlock(height isaac.Height, round isaac.Round, proposal hash.Hash) (isaac.Block, error) {
	if dp.checkers == nil {
		return dp.DefaultProposalValidator.NewBlock(height, round, proposal)
	}

	li, err := condition.NewLogItemFromMap(
		map[string]interface{}{
			"node":  dp.homeState.Home().Alias(),
			"state": dp.homeState.State().String(),
			"block": map[string]interface{}{
				"height":   height.Uint64(),
				"round":    round.Uint64(),
				"proposal": proposal.String(),
			},
		})
	if err != nil {
		return isaac.Block{}, err
	}

	for _, c := range dp.checkers {
		if c.Checker().Check(li) {
			for _, action := range c.Actions() {
				l := dp.Log().With().
					Str("query", c.Checker().Query()).
					Str("action", action.Action()).
					RawJSON("data", li.Bytes()).
					Logger()

				l.Debug().Msg("condition matched")

				switch action.Action() {
				case "fail":
					return isaac.Block{}, xerrors.Errorf("failed to make new block")
				case "random-block-hash":
					newHash := NewRandomBlockHash()
					return isaac.NewBlockWithHash(height, round, proposal, newHash)
				case "block-hash":
					if len(action.Value().Value()) < 1 {
						err := xerrors.Errorf("value not found: %v")
						l.Error().Err(err).Send()
						return isaac.Block{}, err
					} else if action.Value().Hint() != reflect.String {
						err := xerrors.Errorf("invalid value found: %v", action.Value().Hint())
						l.Error().Err(err).Send()
						return isaac.Block{}, err
					}

					newHash, err := hash.NewHashFromString(action.Value().Value()[0].(string))
					if err != nil {
						l.Error().Err(err).Send()
						return isaac.Block{}, err
					}

					return isaac.NewBlockWithHash(height, round, proposal, newHash)
				}
			}
		}
	}

	return dp.DefaultProposalValidator.NewBlock(height, round, proposal)
}
