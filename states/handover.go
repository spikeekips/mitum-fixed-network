package states

import (
	"time"

	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
)

var (
	DefaultPingHandoverInterval = time.Second * 3
	DefaultPassthroughExpire    = DefaultPingHandoverInterval * 2
)

type Handover interface {
	util.Daemon
	UnderHandover() bool
	IsReady() bool
	OldNode() network.Channel
}
