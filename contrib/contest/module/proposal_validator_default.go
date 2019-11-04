package contest_module

import (
	"encoding/json"
	"sync"

	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/isaac"
)

func init() {
	ProposalValidators = append(ProposalValidators, "DefaultProposalValidator")
	ProposalValidatorConfigs["DefaultProposalValidator"] = DefaultProposalValidatorConfig{}
}

type DefaultProposalValidatorConfig struct {
	N string `yaml:"name"`
}

func (cm DefaultProposalValidatorConfig) Name() string {
	return cm.N
}

func (cm *DefaultProposalValidatorConfig) IsValid() error {
	return nil
}

func (cm *DefaultProposalValidatorConfig) Merge(i interface{}) error {
	return nil
}

func (cm DefaultProposalValidatorConfig) New(homeState *isaac.HomeState, l zerolog.Logger) isaac.ProposalValidator {
	cb := NewDefaultProposalValidator(homeState)
	cb.SetLogger(l)

	return cb
}

type DefaultProposalValidator struct {
	*common.Logger
	homeState *isaac.HomeState
	validated *sync.Map
}

func NewDefaultProposalValidator(homeState *isaac.HomeState) DefaultProposalValidator {
	return DefaultProposalValidator{
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "default-proposer_Validator")
		}),
		homeState: homeState,
		validated: &sync.Map{},
	}
}

func (dp DefaultProposalValidator) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type": "DefaultProposalValidator",
	})
}

func (dp DefaultProposalValidator) Validated(proposal hash.Hash) bool {
	_, found := dp.validated.Load(proposal)
	return found
}

func (dp DefaultProposalValidator) NewBlock(height isaac.Height, round isaac.Round, proposal hash.Hash) (isaac.Block, error) {
	if block, found := dp.validated.Load(proposal); found {
		return block.(isaac.Block), nil
	}

	block, err := isaac.NewBlock(height, round, proposal)
	if err != nil {
		return isaac.Block{}, err
	}

	dp.validated.Store(proposal, block)

	return block, nil
}
