package config

type BaseLocalNetworkPackerYAML struct {
	URL       string
	Bind      string
	Cache     string `yaml:"cache,omitempty"`
	SealCache string `yaml:"seal-cache,omitempty"`
}

func (no BaseLocalNetwork) MarshalYAML() (interface{}, error) {
	nno := BaseLocalNetworkPackerYAML{
		URL:  no.URL().String(),
		Bind: no.Bind().String(),
	}

	if no.Cache() != nil {
		nno.Cache = no.Cache().String()
	}

	if no.SealCache() != nil {
		nno.SealCache = no.SealCache().String()
	}

	return nno, nil
}
