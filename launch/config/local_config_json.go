package config

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type BaseLocalConfigJSONPacker struct {
	SyncInterval string `json:"sync-interval,omitempty"`
	TimeServer   string `json:"time_server,omitempty"`
}

func (no DefaultLocalConfig) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BaseLocalConfigJSONPacker{
		SyncInterval: no.syncInterval.String(),
		TimeServer:   no.timeServer,
	})
}
