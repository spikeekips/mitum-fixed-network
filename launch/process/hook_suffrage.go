package process

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type HookHandlerSuffrage func(context.Context, map[string]interface{}) (config.Suffrage, error)

var DefaultHookHandlersSuffrage = map[string]HookHandlerSuffrage{
	"fixed-proposer": SuffrageHandlerFixedProposer,
	"roundrobin":     SuffrageHandlerRoundrobin,
}

func HookSuffrageFunc(handlers map[string]HookHandlerSuffrage) pm.ProcessFunc {
	return func(ctx context.Context) (context.Context, error) {
		var conf config.LocalNode
		var sc map[string]interface{}
		if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
			return nil, err
		} else {
			sc = conf.Source()
		}

		var m map[string]interface{}
		var st string
		if n, err := config.ParseMap(sc, "suffrage", true); err != nil {
			return ctx, err
		} else if n == nil {
			//
		} else if t, err := config.ParseType(n, true); err != nil {
			return ctx, err
		} else {
			st = t
			m = n
		}

		var sf config.Suffrage
		if len(st) < 1 {
			if i, err := SuffrageHandlerRoundrobin(ctx, nil); err != nil {
				return ctx, err
			} else {
				sf = i
			}
		} else if h, found := handlers[st]; !found {
			return nil, xerrors.Errorf("unknown suffrage found, %s", st)
		} else if i, err := h(ctx, m); err != nil {
			return nil, err
		} else {
			sf = i
		}

		if err := conf.SetSuffrage(sf); err != nil {
			return ctx, err
		} else {
			return ctx, nil
		}
	}
}

func SuffrageHandlerFixedProposer(ctx context.Context, m map[string]interface{}) (config.Suffrage, error) {
	var enc *jsonenc.Encoder
	if err := config.LoadJSONEncoderContextValue(ctx, &enc); err != nil {
		return nil, err
	}

	if i, found := m["proposer"]; !found {
		return nil, xerrors.Errorf("proposer not set for fixed-proposer")
	} else if s, ok := i.(string); !ok {
		return nil, xerrors.Errorf("proposer for fixed-proposer should be string, not %T", i)
	} else if address, err := base.DecodeAddressFromString(enc, s); err != nil {
		return nil, xerrors.Errorf("invalid proposer address for fixed-proposer: %w", err)
	} else {
		return config.NewFixedProposerSuffrage(address)
	}
}

func SuffrageHandlerRoundrobin(context.Context, map[string]interface{}) (config.Suffrage, error) {
	return config.NewRoundrobinSuffrage(), nil
}
