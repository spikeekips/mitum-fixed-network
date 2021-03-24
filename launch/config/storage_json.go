package config

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type DatabasePackerJSON struct {
	URI   string
	Cache string
}

type BlockDataPackerJSON struct {
	Path string `json:"path"`
}

type BaseStoragePackerJSON struct {
	Database  DatabasePackerJSON  `json:"database"`
	BlockData BlockDataPackerJSON `json:"blockdata"`
}

func (no BaseStorage) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BaseStoragePackerJSON{
		Database: DatabasePackerJSON{
			URI:   no.database.URI().String(),
			Cache: no.database.Cache().String(),
		},
		BlockData: BlockDataPackerJSON{
			Path: no.blockData.Path(),
		},
	})
}
