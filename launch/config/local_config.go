package config

import (
	"time"
)

var (
	DefaultSyncInterval = time.Second * 10
	DefaultTimeServer   = "time.google.com"
)

type LocalConfig interface {
	SyncInterval() time.Duration
	SetSyncInterval(string) error
	TimeServer() string
	SetTimeServer(string) error
}

type DefaultLocalConfig struct {
	syncInterval time.Duration
	timeServer   string
}

func EmptyDefaultLocalConfig() *DefaultLocalConfig {
	return &DefaultLocalConfig{
		syncInterval: DefaultSyncInterval,
		timeServer:   DefaultTimeServer,
	}
}

func (no *DefaultLocalConfig) SyncInterval() time.Duration {
	return no.syncInterval
}

func (no *DefaultLocalConfig) SetSyncInterval(s string) error {
	if t, err := parseTimeDuration(s, true); err != nil {
		return err
	} else {
		no.syncInterval = t

		return nil
	}
}

func (no *DefaultLocalConfig) TimeServer() string {
	return no.timeServer
}

func (no *DefaultLocalConfig) SetTimeServer(s string) error {
	no.timeServer = s

	return nil
}
