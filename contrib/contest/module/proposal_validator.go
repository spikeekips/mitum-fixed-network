package contest_module

import (
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/isaac"
)

var ProposalValidators []string
var ProposalValidatorConfigs map[string]interface{}

func init() {
	ProposalValidatorConfigs = map[string]interface{}{}
}

type ProposalValidatorConfig interface {
	Name() string
	Merge(interface{}) error
	New(*isaac.HomeState, zerolog.Logger) isaac.ProposalValidator
}
