package config

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type BaseBlockFSPackerJSON struct {
	Path     string `json:"path"`
	WideOpen bool   `json:"wide-open,omitempty"`
}

type BaseStoragePackerJSON struct {
	URI     string                `json:"uri,omitempty"`
	Cache   string                `json:"cache,omitempty"`
	BlockFS BaseBlockFSPackerJSON `json:"blockfs"`
}

func (no BaseStorage) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BaseStoragePackerJSON{
		URI:   no.main.URI().String(),
		Cache: no.main.Cache().String(),
		BlockFS: BaseBlockFSPackerJSON{
			Path:     no.blockFS.Path(),
			WideOpen: no.blockFS.WideOpen(),
		},
	})
}
