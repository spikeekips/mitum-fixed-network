package contest_module

import (
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/isaac"
)

func init() {
	BallotMakers = append(BallotMakers, "DefaultBallotMaker")
	BallotMakerConfigs["DefaultBallotMaker"] = DefaultBallotMakerConfig{}
}

type DefaultBallotMakerConfig struct {
	N string `yaml:"name"`
}

func (pc DefaultBallotMakerConfig) Name() string {
	return pc.N
}

func (pc *DefaultBallotMakerConfig) IsValid() error {
	return nil
}

func (pc *DefaultBallotMakerConfig) Merge(interface{}) error {
	return nil
}

func (pc DefaultBallotMakerConfig) New(homeState *isaac.HomeState, l zerolog.Logger) isaac.BallotMaker {
	bm := isaac.NewDefaultBallotMaker(homeState.Home())
	bm.SetLogger(l)

	return bm
}
