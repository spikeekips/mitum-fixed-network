package process

import (
	"context"
	"crypto/tls"
	"net/url"
	"time"

	"golang.org/x/xerrors"

	"github.com/lucas-clemente/quic-go"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
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
	var conf config.LocalNetwork
	if err := config.LoadConfigContextValue(ctx, &ln); err != nil {
		return ctx, err
	} else {
		conf = ln.Network()
	}

	var encs *encoder.Encoders
	if err := config.LoadEncodersContextValue(ctx, &encs); err != nil {
		return ctx, err
	}

	var l logging.Logger
	if err := config.LoadLogContextValue(ctx, &l); err != nil {
		return ctx, err
	}

	var ca cache.Cache
	if c, err := cache.NewCacheFromURI(conf.Cache().String()); err != nil {
		return ctx, err
	} else {
		ca = c
	}

	if nt, err := NewNetworkServer(conf.Bind().Host, conf.URL(), encs, ca); err != nil {
		return ctx, err
	} else {
		if i, ok := nt.(logging.SetLogger); ok {
			_ = i.SetLogger(l)
		}

		ctx = context.WithValue(ctx, ContextValueNetwork, nt)

		return ctx, nil
	}
}

func NewNetworkServer(bind string, u *url.URL, encs *encoder.Encoders, ca cache.Cache) (network.Server, error) {
	var je encoder.Encoder
	if e, err := encs.Encoder(jsonenc.JSONType, ""); err != nil {
		return nil, xerrors.Errorf("json encoder needs for quic-network: %w", err)
	} else {
		je = e
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

func LoadNodeChannel(
	u *url.URL,
	encs *encoder.Encoders,
	connectionTimeout time.Duration,
	inseucre bool,
) (network.Channel, error) {
	var je encoder.Encoder
	if e, err := encs.Encoder(jsonenc.JSONType, ""); err != nil {
		return nil, xerrors.Errorf("json encoder needs for quic-network: %w", err)
	} else {
		je = e
	}

	switch u.Scheme {
	case "quic":
		quicConfig := &quic.Config{HandshakeTimeout: connectionTimeout}
		if ch, err := quicnetwork.NewChannel(
			u.String(),
			100,
			inseucre,
			quicConfig,
			encs,
			je,
		); err != nil {
			return nil, err
		} else {
			return ch, nil
		}
	default:
		return nil, xerrors.Errorf("not supported publish URL, %v", u.String())
	}
}
