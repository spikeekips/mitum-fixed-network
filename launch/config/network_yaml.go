package config

type BaseLocalNetworkPackerYAML struct {
	URL  string
	Bind string
}

func (no BaseLocalNetwork) MarshalYAML() (interface{}, error) {
	return BaseLocalNetworkPackerYAML{
		URL:  no.URL().String(),
		Bind: no.Bind().String(),
	}, nil
}
