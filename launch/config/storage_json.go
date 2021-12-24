package config

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type DatabasePackerJSON struct {
	URI   string
	Cache string
}

type BlockdataPackerJSON struct {
	Path string `json:"path"`
}

type BaseStoragePackerJSON struct {
	Database  DatabasePackerJSON  `json:"database"`
	Blockdata BlockdataPackerJSON `json:"blockdata"`
}

func (no BaseStorage) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BaseStoragePackerJSON{
		Database: DatabasePackerJSON{
			URI:   no.database.URI().String(),
			Cache: no.database.Cache().String(),
		},
		Blockdata: BlockdataPackerJSON{
			Path: no.blockdata.Path(),
		},
	})
}
