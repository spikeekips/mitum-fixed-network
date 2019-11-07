package contest_module

import (
	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/isaac"
)

var BallotMakers []string
var BallotMakerConfigs map[string]interface{}

func init() {
	BallotMakerConfigs = map[string]interface{}{}
}

type BallotMakerConfig interface {
	Name() string
	Merge(interface{}) error
	New(*isaac.HomeState, zerolog.Logger) isaac.BallotMaker
}
