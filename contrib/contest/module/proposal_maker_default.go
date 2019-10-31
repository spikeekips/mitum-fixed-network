package contest_module

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/isaac"
	"golang.org/x/xerrors"
)

func init() {
	ProposalMakers = append(ProposalMakers, "DefaultProposalMaker")
	ProposalMakerConfigs["DefaultProposalMaker"] = DefaultProposalMakerConfig{}
}

type DefaultProposalMakerConfig struct {
	N string        `yaml:"name"`
	D time.Duration `yaml:"delay,omitempty"`
}

func (pc DefaultProposalMakerConfig) Name() string {
	return pc.N
}

func (pc DefaultProposalMakerConfig) Delay() time.Duration {
	return pc.D
}

func (pc *DefaultProposalMakerConfig) IsValid() error {
	return nil
}

func (pc *DefaultProposalMakerConfig) Merge(i interface{}) error {
	n, ok := interface{}(i).(ProposalMakerConfig)
	if !ok {
		return xerrors.Errorf("invalid merge source found: %%", i)
	}

	if pc.D < 1 {
		pc.D = n.Delay()
	}

	return nil
}

func (pc DefaultProposalMakerConfig) New(homeState *isaac.HomeState, l zerolog.Logger) isaac.ProposalMaker {
	pm := isaac.NewDefaultProposalMaker(homeState.Home(), pc.Delay())
	pm.SetLogger(l)

	return pm
}
