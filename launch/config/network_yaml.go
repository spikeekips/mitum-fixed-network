package config

type BaseLocalNetworkPackerYAML struct {
	URL       string
	Bind      string
	Cache     string
	SealCache string `yaml:"seal-cache,omitempty"`
}

func (no BaseLocalNetwork) MarshalYAML() (interface{}, error) {
	return BaseLocalNetworkPackerYAML{
		URL:       no.URL().String(),
		Bind:      no.Bind().String(),
		Cache:     no.Cache().String(),
		SealCache: no.SealCache().String(),
	}, nil
}
