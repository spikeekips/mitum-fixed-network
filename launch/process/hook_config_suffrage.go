package process

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type HookHandlerSuffrageConfig func(context.Context, map[string]interface{}, []base.Address) (config.Suffrage, error)

var DefaultHookHandlersSuffrageConfig = map[string]HookHandlerSuffrageConfig{
	"fixed-suffrage": SuffrageConfigHandlerFixedProposer,
	"roundrobin":     SuffrageConfigHandlerRoundrobin,
}

func HookSuffrageConfigFunc(handlers map[string]HookHandlerSuffrageConfig) pm.ProcessFunc {
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

		var nodes []base.Address
		if i, err := parseSuffrageNodes(ctx, m); err != nil {
			return ctx, err
		} else {
			nodes = i
		}

		var sf config.Suffrage
		if len(st) < 1 {
			if i, err := SuffrageConfigHandlerRoundrobin(ctx, nil, nodes); err != nil {
				return ctx, err
			} else {
				sf = i
			}
		} else if h, found := handlers[st]; !found {
			return nil, xerrors.Errorf("unknown suffrage found, %s", st)
		} else if i, err := h(ctx, m, nodes); err != nil {
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

func SuffrageConfigHandlerFixedProposer(
	ctx context.Context,
	m map[string]interface{},
	nodes []base.Address,
) (config.Suffrage, error) {
	var enc *jsonenc.Encoder
	if err := config.LoadJSONEncoderContextValue(ctx, &enc); err != nil {
		return nil, err
	}

	var numberOfActing uint
	if i, found := m["number-of-acting"]; !found {
		numberOfActing = isaac.DefaultPolicyNumberOfActingSuffrageNodes
	} else {
		switch n := i.(type) {
		case int:
			numberOfActing = uint(n)
		case uint:
			numberOfActing = n
		default:
			return nil, xerrors.Errorf("invalid type for number-of-acting, %T", i)
		}
	}

	var proposer base.Address
	if i, found := m["proposer"]; found {
		if a, err := parseAddress(i, enc); err != nil {
			return nil, xerrors.Errorf("invalid proposer address for fixed-suffrage: %w", err)
		} else {
			proposer = a
		}
	}

	if proposer == nil {
		return nil, xerrors.Errorf("empty proposer")
	}

	return config.NewFixedSuffrage(proposer, nodes, numberOfActing), nil
}

func SuffrageConfigHandlerRoundrobin(
	_ context.Context,
	m map[string]interface{},
	nodes []base.Address,
) (config.Suffrage, error) {
	var numberOfActing uint
	if i, found := m["number-of-acting"]; !found {
		numberOfActing = isaac.DefaultPolicyNumberOfActingSuffrageNodes
	} else {
		switch n := i.(type) {
		case int:
			numberOfActing = uint(n)
		case uint:
			numberOfActing = n
		default:
			return nil, xerrors.Errorf("invalid type for number-of-acting, %T", i)
		}
	}

	return config.NewRoundrobinSuffrage(nodes, numberOfActing), nil
}

func parseSuffrageNodes(ctx context.Context, m map[string]interface{}) ([]base.Address, error) {
	var enc *jsonenc.Encoder
	if err := config.LoadJSONEncoderContextValue(ctx, &enc); err != nil {
		return nil, err
	}

	var l []interface{}
	if i, found := m["nodes"]; !found {
		return nil, xerrors.Errorf("nodes not found in suffrage config")
	} else if j, ok := i.([]interface{}); !ok {
		return nil, xerrors.Errorf("invalid nodes list, %T", i)
	} else {
		l = j
	}

	nodes := make([]base.Address, len(l))
	for j := range l {
		if a, err := parseAddress(l[j], enc); err != nil {
			return nil, xerrors.Errorf("invalid node address for suffrage config: %w", err)
		} else {
			nodes[j] = a
		}
	}

	return nodes, nil
}

func parseAddress(i interface{}, enc *jsonenc.Encoder) (base.Address, error) {
	if s, ok := i.(string); !ok {
		return nil, xerrors.Errorf("not address string, not %T", i)
	} else if address, err := base.DecodeAddressFromString(enc, s); err != nil {
		return nil, xerrors.Errorf("invalid address: %w", err)
	} else if err := address.IsValid(nil); err != nil {
		return nil, xerrors.Errorf("invalid address: %w", err)
	} else {
		return address, err
	}
}
