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

func (cm DefaultProposalValidatorConfig) New(homeState *isaac.HomeState, sealStorage isaac.SealStorage, l zerolog.Logger) isaac.ProposalValidator {
	cb := NewDefaultProposalValidator(homeState, sealStorage)
	cb.SetLogger(l)

	return cb
}

type DefaultProposalValidator struct {
	*common.Logger
	isaac.BaseProposalValidator
	homeState *isaac.HomeState
	validated *sync.Map
}

func NewDefaultProposalValidator(homeState *isaac.HomeState, sealStorage isaac.SealStorage) DefaultProposalValidator {
	return DefaultProposalValidator{
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "default-proposer_Validator")
		}),
		BaseProposalValidator: isaac.NewBaseProposalValidator(sealStorage),
		homeState:             homeState,
		validated:             &sync.Map{},
	}
}

func (dp DefaultProposalValidator) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type": "DefaultProposalValidator",
	})
}

func (dp DefaultProposalValidator) Validated(h hash.Hash) bool {
	_, found := dp.validated.Load(h)
	return found
}

func (dp DefaultProposalValidator) SetNewBlock(block isaac.Block) {
	dp.validated.Store(block.Proposal(), block)
}

func (dp DefaultProposalValidator) NewBlock(h hash.Hash) (isaac.Block, error) {
	if block, found := dp.validated.Load(h); found {
		return block.(isaac.Block), nil
	}

	proposal, err := dp.BaseProposalValidator.GetProposal(h)
	if err != nil {
		return isaac.Block{}, err
	}

	block, err := isaac.NewBlock(proposal.Height(), proposal.Round(), proposal.Hash())
	if err != nil {
		return isaac.Block{}, err
	}

	dp.validated.Store(proposal.Hash(), block)

	return block, nil
}
