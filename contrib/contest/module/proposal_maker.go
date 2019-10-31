package contest_module

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/isaac"
)

var ProposalMakers []string
var ProposalMakerConfigs map[string]interface{}

func init() {
	ProposalMakerConfigs = map[string]interface{}{}
}

type ProposalMakerConfig interface {
	Name() string
	Delay() time.Duration
	Merge(interface{}) error
	New(*isaac.HomeState, zerolog.Logger) isaac.ProposalMaker
}
