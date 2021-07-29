package yamlconfig

import (
	"context"

	"github.com/spikeekips/mitum/launch/config"
	"golang.org/x/xerrors"
)

type NodeNetwork struct {
	Extras map[string]interface{} `yaml:",inline"`
}

type LocalNetwork struct {
	Bind        *string                `yaml:"bind"`
	URL         *string                `yaml:"url"`
	CertKeyFile *string                `yaml:"cert-key,omitempty"`
	CertFile    *string                `yaml:"cert,omitempty"`
	Cache       *string                `yaml:",omitempty"`
	SealCache   *string                `yaml:"seal-cache,omitempty"`
	RateLimit   *RateLimit             `yaml:"rate-limit,omitempty"`
	Extras      map[string]interface{} `yaml:",inline"`
}

func (no LocalNetwork) Set(ctx context.Context) (context.Context, error) {
	var l config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	}
	conf := l.Network()

	if no.URL != nil {
		if err := conf.SetURL(*no.URL); err != nil {
			return ctx, err
		}
	}

	if no.Bind != nil {
		if err := conf.SetBind(*no.Bind); err != nil {
			return ctx, err
		}
	}

	if (no.CertKeyFile != nil || no.CertFile != nil) && (no.CertKeyFile == nil || no.CertFile == nil) {
		return ctx, xerrors.Errorf("cert-key and cert should be given both")
	} else if no.CertKeyFile != nil {
		if err := conf.SetCertFiles(*no.CertFile, *no.CertKeyFile); err != nil {
			return ctx, err
		}
	}

	if no.Cache != nil {
		if err := conf.SetCache(*no.Cache); err != nil {
			return ctx, err
		}
	}

	if no.SealCache != nil {
		if err := conf.SetSealCache(*no.SealCache); err != nil {
			return ctx, err
		}
	}

	if no.RateLimit != nil {
		i, err := no.RateLimit.Set(ctx)
		if err != nil {
			return ctx, err
		}
		ctx = i
	}

	return ctx, nil
}
