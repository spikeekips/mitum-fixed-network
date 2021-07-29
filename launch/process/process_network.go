package process

import (
	"context"
	"crypto/tls"
	"net/url"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/network/discovery"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/util"
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
			ProcessNameConsensusStates,
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

	var l logging.Logger
	if err := config.LoadNetworkLogContextValue(ctx, &l); err != nil {
		return ctx, err
	}

	ca, err := cache.NewCacheFromURI(conf.Cache().String())
	if err != nil {
		return ctx, err
	}

	nt, err := NewNetworkServer(conf.Bind().Host, conf.ConnInfo().URL(), encs, ca)
	if err != nil {
		return ctx, err
	}
	if i, ok := nt.(logging.SetLogger); ok {
		_ = i.SetLogger(l)
	}

	ctx = context.WithValue(ctx, ContextValueNetwork, nt)

	return ctx, nil
}

func NewNetworkServer(bind string, u *url.URL, encs *encoder.Encoders, ca cache.Cache) (network.Server, error) {
	je, err := encs.Encoder(jsonenc.JSONEncoderType, "")
	if err != nil {
		return nil, xerrors.Errorf("json encoder needs for quic-network: %w", err)
	}

	var certs []tls.Certificate
	if priv, err := util.GenerateED25519Privatekey(); err != nil {
		return nil, err
	} else if ct, err := util.GenerateTLSCerts(u.Host, priv); err != nil {
		return nil, err
	} else {
		certs = ct
	}

	if qs, err := quicnetwork.NewPrimitiveQuicServer(bind, certs); err != nil {
		return nil, err
	} else if nqs, err := quicnetwork.NewServer(qs, encs, je, ca); err != nil {
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
