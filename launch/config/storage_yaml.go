package config

type BaseBlockFSPackerYAML struct {
	Path     string
	WideOpen bool `yaml:"wide-open,omitempty"`
}

type BaseStoragePackerYAML struct {
	URI     string                `yaml:",omitempty"`
	Cache   string                `yaml:",omitempty"`
	BlockFS BaseBlockFSPackerYAML `yaml:"blockfs"`
}

func (no BaseStorage) MarshalYAML() (interface{}, error) {
	return BaseStoragePackerYAML{
		URI:   no.main.URI().String(),
		Cache: no.main.Cache().String(),
		BlockFS: BaseBlockFSPackerYAML{
			Path:     no.blockFS.Path(),
			WideOpen: no.blockFS.WideOpen(),
		},
	}, nil
}
