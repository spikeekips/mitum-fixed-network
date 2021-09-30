package process

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/network/discovery"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
)

const ProcessNameNetwork = "network"

var ProcessorNetwork pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameNetwork,
		[]string{
			ProcessNameConfig,
		},
		ProcessQuicNetwork,
	); err != nil {
		panic(err)
	} else {
		ProcessorNetwork = i
	}
}

func ProcessQuicNetwork(ctx context.Context) (context.Context, error) {
	var ln config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &ln); err != nil {
		return ctx, err
	}
	conf := ln.Network()

	var encs *encoder.Encoders
	if err := config.LoadEncodersContextValue(ctx, &encs); err != nil {
		return ctx, err
	}

	var nodepool *network.Nodepool
	if err := LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return ctx, err
	}

	var l *logging.Logging
	if err := config.LoadLogContextValue(ctx, &l); err != nil {
		return ctx, err
	}

	var httpLog *logging.Logging
	if err := config.LoadNetworkLogContextValue(ctx, &httpLog); err != nil {
		return ctx, err
	}

	ca, err := cache.NewCacheFromURI(conf.Cache().String())
	if err != nil {
		return ctx, err
	}

	nt, err := NewNetworkServer(conf.Bind().Host, conf.Certs(), encs, ca, conf.ConnInfo(), nodepool, httpLog)
	if err != nil {
		return ctx, err
	}
	if i, ok := nt.(logging.SetLogging); ok {
		_ = i.SetLogging(l)
	}

	ctx = context.WithValue(ctx, ContextValueNetwork, nt)

	return ctx, nil
}

func NewNetworkServer(
	bind string,
	certs []tls.Certificate,
	encs *encoder.Encoders,
	ca cache.Cache,
	connInfo network.ConnInfo,
	nodepool *network.Nodepool,
	httpLog *logging.Logging,
) (network.Server, error) {
	je, err := encs.Encoder(jsonenc.JSONEncoderType, "")
	if err != nil {
		return nil, errors.Wrap(err, "json encoder needs for quic-network")
	}

	if qs, err := quicnetwork.NewPrimitiveQuicServer(bind, certs, httpLog); err != nil {
		return nil, err
	} else if nqs, err := quicnetwork.NewServer(qs, encs, je, ca, connInfo, nodepool.Passthroughs); err != nil {
		return nil, err
	} else if err := nqs.Initialize(); err != nil {
		return nil, err
	} else {
		return nqs, nil
	}
}

func LoadNodeChannel( // TODO remove
	connInfo network.ConnInfo,
	encs *encoder.Encoders,
	connectionTimeout time.Duration,
) (network.Channel, error) {
	return discovery.LoadNodeChannel(connInfo, encs, connectionTimeout)
}
