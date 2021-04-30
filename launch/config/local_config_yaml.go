package config

import "time"

type BaseLocalConfigYAMLPacker struct {
	SyncInterval time.Duration `yaml:"sync-interval,omitempty"`
	TimeServer   string        `yaml:"time-server,omitempty"`
}

func (no DefaultLocalConfig) MarshalYAML() (interface{}, error) {
	return BaseLocalConfigYAMLPacker{
		SyncInterval: no.syncInterval,
		TimeServer:   no.timeServer,
	}, nil
}
