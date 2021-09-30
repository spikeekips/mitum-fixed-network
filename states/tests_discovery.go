// go:build test
//go:build test
// +build test

package states

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

func NewTestDiscoveryJoiner() *DiscoveryJoiner {
	return &DiscoveryJoiner{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "states-discovery")
		}),
		ij:               util.NewLockedItem(false),
		keeptryingCancel: func() {},
	}
}

func (sd *DiscoveryJoiner) SetJoined(b bool) {
	_ = sd.ij.Set(b)
}

func (sd *DiscoveryJoiner) SetJoinFunc(f func() error) {
	sd.joinfunc = f
}

func (sd *DiscoveryJoiner) SetLeaveFunc(f func(time.Duration) error) {
	sd.leaveFunc = f
}
