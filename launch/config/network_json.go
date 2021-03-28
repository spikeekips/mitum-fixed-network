package config

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type BaseLocalNetworkPackerJSON struct {
	URL       string `json:"url"`
	Bind      string `json:"bind"`
	Cache     string `json:"cache,omitempty"`
	SealCache string `json:"seal_cache,omitempty"`
}

func (no BaseLocalNetwork) MarshalJSON() ([]byte, error) {
	nno := BaseLocalNetworkPackerJSON{
		URL:  no.URL().String(),
		Bind: no.Bind().String(),
	}
	if no.Cache() != nil {
		nno.Cache = no.Cache().String()
	}

	if no.SealCache() != nil {
		nno.SealCache = no.SealCache().String()
	}

	return jsonenc.Marshal(nno)
}
