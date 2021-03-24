package config

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type BaseBlockDataPackerJSON struct {
	Path string `json:"path"`
}

type BaseStoragePackerJSON struct {
	URI       string                  `json:"uri,omitempty"`
	Cache     string                  `json:"cache,omitempty"`
	BlockData BaseBlockDataPackerJSON `json:"blockdata"`
}

func (no BaseStorage) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BaseStoragePackerJSON{
		URI:   no.main.URI().String(),
		Cache: no.main.Cache().String(),
		BlockData: BaseBlockDataPackerJSON{
			Path: no.blockData.Path(),
		},
	})
}
