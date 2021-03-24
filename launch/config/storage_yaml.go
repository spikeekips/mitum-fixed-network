package config

type BaseBlockDataPackerYAML struct {
	Path string
}

type BaseStoragePackerYAML struct {
	URI       string                  `yaml:",omitempty"`
	Cache     string                  `yaml:",omitempty"`
	BlockData BaseBlockDataPackerYAML `yaml:"blockdata"`
}

func (no BaseStorage) MarshalYAML() (interface{}, error) {
	return BaseStoragePackerYAML{
		URI:   no.main.URI().String(),
		Cache: no.main.Cache().String(),
		BlockData: BaseBlockDataPackerYAML{
			Path: no.blockData.Path(),
		},
	}, nil
}
