package config

type DatabasePackerYAML struct {
	URI   string `yaml:",omitempty"`
	Cache string `yaml:",omitempty"`
}

type BlockDataPackerYAML struct {
	Path string
}

type BaseStoragePackerYAML struct {
	Database  DatabasePackerYAML  `yaml:"database"`
	BlockData BlockDataPackerYAML `yaml:"blockdata"`
}

func (no BaseStorage) MarshalYAML() (interface{}, error) {
	return BaseStoragePackerYAML{
		Database: DatabasePackerYAML{
			URI:   no.database.URI().String(),
			Cache: no.database.Cache().String(),
		},
		BlockData: BlockDataPackerYAML{
			Path: no.blockData.Path(),
		},
	}, nil
}
