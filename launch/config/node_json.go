package config

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type BaseRemoteNodePackerJSON struct {
	Address   base.Address  `json:"address"`
	URL       string        `json:"url"`
	Publickey key.Publickey `json:"publickey"`
}

func (no BaseRemoteNode) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BaseRemoteNodePackerJSON{
		Address:   no.Address(),
		URL:       no.URL().String(),
		Publickey: no.Publickey(),
	})
}
