package yamlconfig

import (
	"context"
	"strings"

	"github.com/spikeekips/mitum/launch/config"
)

type LocalConfig struct {
	SyncInterval *string `yaml:"sync-interval"`
	TimeServer   *string `yaml:"time-server,omitempty"`
}

func (no LocalConfig) Set(ctx context.Context) (context.Context, error) {
	var l config.LocalNode
	var conf config.LocalConfig
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	} else {
		conf = l.LocalConfig()
	}

	if no.TimeServer != nil {
		if err := conf.SetTimeServer(strings.TrimSpace(*no.TimeServer)); err != nil {
			return ctx, err
		}
	}

	if no.SyncInterval != nil {
		if err := conf.SetSyncInterval(*no.SyncInterval); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}
