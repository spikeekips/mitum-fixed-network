package yamlconfig

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/launch/config"
)

type LocalNode struct {
	Node              `yaml:",inline"`
	NetworkID         *string                  `yaml:"network-id,omitempty"`
	Privatekey        *string                  `yaml:",omitempty"`
	Network           *LocalNetwork            `yaml:",omitempty"`
	Storage           *Storage                 `yaml:",omitempty"`
	Nodes             []*RemoteNode            `yaml:",omitempty"`
	Suffrage          map[string]interface{}   `yaml:"suffrage,omitempty"`
	ProposalProcessor map[string]interface{}   `yaml:"proposal-processor,omitempty"`
	Policy            *Policy                  `yaml:",omitempty"`
	GenesisOperations []map[string]interface{} `yaml:"genesis-operations,omitempty"`
	LocalConfig       *LocalConfig             `yaml:",inline"`
	Extras            map[string]interface{}   `yaml:",inline"`
}

func (no LocalNode) Set(ctx context.Context) (context.Context, error) {
	for _, f := range []func() error{
		no.checkSuffrage,
		no.checkProposalProcessor,
		no.checkGenesisOperations,
	} {
		if err := f(); err != nil {
			return ctx, err
		}
	}

	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return ctx, err
	}

	for _, f := range []func(context.Context, config.LocalNode) (context.Context, error){
		no.setBase,
		no.setComponents,
		no.setNodes,
	} {
		c, err := f(ctx, conf)
		if err != nil {
			return ctx, err
		}
		ctx = c
	}

	return ctx, nil
}

func (no LocalNode) setBase(ctx context.Context, conf config.LocalNode) (context.Context, error) {
	if no.Address != nil {
		if err := conf.SetAddress(*no.Address); err != nil {
			return ctx, err
		}
	}

	if no.Privatekey != nil {
		if err := conf.SetPrivatekey(*no.Privatekey); err != nil {
			return ctx, err
		}
	}

	if no.NetworkID != nil {
		if err := conf.SetNetworkID(*no.NetworkID); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}

func (no LocalNode) setComponents(ctx context.Context, _ config.LocalNode) (context.Context, error) {
	if no.Network != nil {
		c, err := no.Network.Set(ctx)
		if err != nil {
			return ctx, err
		}
		ctx = c
	}

	if no.Storage != nil {
		c, err := no.Storage.Set(ctx)
		if err != nil {
			return ctx, err
		}
		ctx = c
	}

	if no.Policy != nil {
		c, err := no.Policy.Set(ctx)
		if err != nil {
			return ctx, err
		}
		ctx = c
	}

	if no.LocalConfig != nil {
		c, err := no.LocalConfig.Set(ctx)
		if err != nil {
			return ctx, err
		}
		ctx = c
	}

	return ctx, nil
}

func (no LocalNode) setNodes(ctx context.Context, conf config.LocalNode) (context.Context, error) {
	if len(no.Nodes) < 1 {
		if err := conf.SetNodes(nil); err != nil {
			return ctx, err
		}

		return ctx, nil
	}

	nodes := make([]config.RemoteNode, len(no.Nodes))
	for i := range no.Nodes {
		c, err := no.Nodes[i].Load(ctx)
		if err != nil {
			return ctx, err
		}
		nodes[i] = c
	}

	if err := conf.SetNodes(nodes); err != nil {
		return ctx, err
	}

	return ctx, nil
}

func (no LocalNode) checkProposalProcessor() error {
	if no.ProposalProcessor == nil {
		return nil
	}

	if s, found := no.ProposalProcessor["type"]; !found {
		return errors.Errorf("'type' is missing in proposal_processor")
	} else if t, ok := s.(string); !ok {
		return errors.Errorf("invalie 'type' type, %T", s)
	} else if len(strings.TrimSpace(t)) < 1 {
		return errors.Errorf("empty 'type'")
	}

	return nil
}

func (no LocalNode) checkSuffrage() error {
	if no.Suffrage == nil {
		return nil
	}

	if _, found := no.Suffrage["nodes"]; !found {
		return errors.Errorf("'nodes' is missing in suffrage")
	}

	if s, found := no.Suffrage["type"]; found {
		if t, ok := s.(string); !ok {
			return errors.Errorf("invalie 'type' type, %T", s)
		} else if len(strings.TrimSpace(t)) < 1 {
			return errors.Errorf("empty 'type'")
		}
	}

	return nil
}

func (no LocalNode) checkGenesisOperations() error {
	for _, v := range no.GenesisOperations {
		if s, found := v["type"]; !found {
			return errors.Errorf("'type' is missing in operation")
		} else if t, ok := s.(string); !ok {
			return errors.Errorf("invalie 'type' type, %T", s)
		} else if len(strings.TrimSpace(t)) < 1 {
			return errors.Errorf("empty 'type'")
		}
	}

	return nil
}
