package discovery

import (
	"sync"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/network"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
)

type NodepoolDelegate struct {
	sync.Mutex
	*logging.Logging
	nodepool          *network.Nodepool
	encs              *encoder.Encoders
	connectionTimeout time.Duration
}

func NewNodepoolDelegate(
	nodepool *network.Nodepool,
	encs *encoder.Encoders,
	connectionTimeout time.Duration,
) *NodepoolDelegate {
	return &NodepoolDelegate{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "discovery-nodepool-delegate")
		}),
		nodepool:          nodepool,
		encs:              encs,
		connectionTimeout: connectionTimeout,
	}
}

func (dg *NodepoolDelegate) NotifyJoin(ci NodeConnInfo) {
	go dg.notifyJoin(ci)
}

func (dg *NodepoolDelegate) notifyJoin(ci NodeConnInfo) {
	dg.Lock()
	defer dg.Unlock()

	addr := ci.Node()
	l := dg.Log().With().Str("notify", "join").Stringer("node", addr).Interface("conninfo", ci).Logger()

	if !dg.nodepool.Exists(addr) {
		l.Error().Msg("unknown node")

		return
	} else if addr.Equal(dg.nodepool.LocalNode().Address()) {
		l.Debug().Msg("local node will be ignored")

		return
	}

	ch, err := dg.channel(ci)
	if err != nil {
		l.Error().Err(err).Msg("failed to make channel")

		return
	}

	if err := dg.nodepool.SetChannel(addr, ch); err != nil {
		l.Error().Err(err).Msg("failed to set channel")

		return
	}

	l.Debug().Msg("node channel updated")
}

func (dg *NodepoolDelegate) NotifyLeave(ci NodeConnInfo, lefts []NodeConnInfo) {
	go dg.notifyLeave(ci, lefts)
}

func (dg *NodepoolDelegate) notifyLeave(ci NodeConnInfo, lefts []NodeConnInfo) {
	dg.Lock()
	defer dg.Unlock()

	addr := ci.Node()
	l := dg.Log().With().
		Str("notify", "leave").Stringer("node", addr).Interface("conninfo", ci).Interface("lefts", lefts).
		Logger()

	if !dg.nodepool.Exists(addr) {
		l.Error().Msg("unknown node")

		return
	} else if addr.Equal(dg.nodepool.LocalNode().Address()) {
		l.Debug().Msg("local node will be ignored")

		return
	}

	if len(lefts) < 1 {
		if err := dg.nodepool.SetChannel(addr, nil); err != nil {
			l.Error().Err(err).Msg("failed to set channel to nil")

			return
		}

		l.Debug().Msg("node channel set to nil")

		return
	}

	updated := lefts[0]
	if !dg.isNewConnInfo(updated) {
		l.Debug().Msg("nothing to update")

		return
	}

	ch, err := dg.channel(updated)
	if err != nil {
		l.Error().Err(err).Msg("failed to make channel")

		return
	}

	if err := dg.nodepool.SetChannel(addr, ch); err != nil {
		l.Error().Err(err).Msg("failed to set channel to nil")

		return
	}

	l.Debug().Msg("node channel updated")
}

func (dg *NodepoolDelegate) NotifyUpdate(ci NodeConnInfo) {
	go dg.notifyUpdate(ci)
}

func (dg *NodepoolDelegate) notifyUpdate(ci NodeConnInfo) {
	dg.Lock()
	defer dg.Unlock()

	addr := ci.Node()
	l := dg.Log().With().Str("notify", "update").Stringer("node", addr).Interface("conninfo", ci).Logger()

	if !dg.nodepool.Exists(addr) {
		l.Error().Msg("unknown node")

		return
	} else if addr.Equal(dg.nodepool.LocalNode().Address()) {
		l.Debug().Msg("local node will be ignored")

		return
	}

	if !dg.isNewConnInfo(ci) {
		l.Debug().Msg("nothing to update")

		return
	}

	ch, err := dg.channel(ci)
	if err != nil {
		l.Error().Err(err).Msg("failed to make channel")

		return
	}

	if err := dg.nodepool.SetChannel(addr, ch); err != nil {
		l.Error().Err(err).Msg("failed to set channel to nil")

		return
	}

	l.Debug().Msg("node channel updated")
}

func (dg *NodepoolDelegate) channel(ci NodeConnInfo) (network.Channel, error) {
	return LoadNodeChannel(ci, dg.encs, dg.connectionTimeout)
}

func (dg *NodepoolDelegate) isNewConnInfo(ci NodeConnInfo) bool {
	switch _, ch, _ := dg.nodepool.Node(ci.Node()); {
	case ch == nil:
		return true
	default:
		b := ch.ConnInfo()
		if ci.URL().String() != b.URL().String() && ci.Insecure() != b.Insecure() {
			return true
		}

		return false
	}
}

func LoadNodeChannel(
	connInfo network.ConnInfo,
	encs *encoder.Encoders,
	connectionTimeout time.Duration,
) (network.Channel, error) {
	if err := connInfo.IsValid(nil); err != nil {
		return nil, err
	}

	if connInfo.URL() == nil {
		return nil, errors.Errorf("connInfo has nil URL, %v", connInfo)
	}

	je, err := encs.Encoder(jsonenc.JSONEncoderType, "")
	if err != nil {
		return nil, errors.Wrap(err, "json encoder needs for quic-network")
	}

	switch connInfo.URL().Scheme {
	case "https":
		quicConfig := &quic.Config{HandshakeIdleTimeout: connectionTimeout}
		ch, err := quicnetwork.NewChannel(
			connInfo,
			100,
			quicConfig,
			encs,
			je,
		)
		if err != nil {
			return nil, err
		}
		return ch, nil
	default:
		return nil, errors.Errorf("not supported publish URL, %v", connInfo)
	}
}
