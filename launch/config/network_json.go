package config

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type BaseLocalNetworkPackerJSON struct {
	URL       string `json:"url"`
	Bind      string `json:"bind"`
	Cache     string `json:"cache"`
	SealCache string `json:"seal_cache"`
}

func (no BaseLocalNetwork) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BaseLocalNetworkPackerJSON{
		URL:       no.URL().String(),
		Bind:      no.Bind().String(),
		Cache:     no.Cache().String(),
		SealCache: no.SealCache().String(),
	})
}
