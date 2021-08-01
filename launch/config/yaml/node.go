package yamlconfig

import (
	"context"

	"github.com/spikeekips/mitum/launch/config"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type Node struct {
	Address *string `yaml:",omitempty"`
}

type RemoteNode struct {
	Node        `yaml:",inline"`
	Publickey   *string                `yaml:",omitempty"`
	URL         *string                `yaml:"url,omitempty"`
	TLSInsecure *bool                  `yaml:"tls-insecure,omitempty"`
	Extras      map[string]interface{} `yaml:",inline"`
}

func (no RemoteNode) Load(ctx context.Context) (config.RemoteNode, error) {
	var enc *jsonenc.Encoder
	if err := config.LoadJSONEncoderContextValue(ctx, &enc); err != nil {
		return nil, err
	}
	conf := config.NewBaseRemoteNode(enc)

	if no.Address != nil {
		if err := conf.SetAddress(*no.Address); err != nil {
			return nil, err
		}
	}

	if no.Publickey != nil {
		if err := conf.SetPublickey(*no.Publickey); err != nil {
			return nil, err
		}
	}

	if no.URL != nil {
		var insecure bool
		if no.TLSInsecure != nil {
			insecure = *no.TLSInsecure
		}

		if err := conf.SetConnInfo(*no.URL, insecure); err != nil {
			return nil, err
		}
	}

	return conf, nil
}
