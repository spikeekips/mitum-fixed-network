package config

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type BaseLocalNetworkPackerJSON struct {
	URL  string `json:"url"`
	Bind string `json:"bind"`
}

func (no BaseLocalNetwork) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BaseLocalNetworkPackerJSON{
		URL:  no.URL().String(),
		Bind: no.Bind().String(),
	})
}
