package config

type DatabasePackerYAML struct {
	URI   string `yaml:",omitempty"`
	Cache string `yaml:",omitempty"`
}

type BlockdataPackerYAML struct {
	Path string
}

type BaseStoragePackerYAML struct {
	Database  DatabasePackerYAML  `yaml:"database"`
	Blockdata BlockdataPackerYAML `yaml:"blockdata"`
}

func (no BaseStorage) MarshalYAML() (interface{}, error) {
	return BaseStoragePackerYAML{
		Database: DatabasePackerYAML{
			URI:   no.database.URI().String(),
			Cache: no.database.Cache().String(),
		},
		Blockdata: BlockdataPackerYAML{
			Path: no.blockdata.Path(),
		},
	}, nil
}
