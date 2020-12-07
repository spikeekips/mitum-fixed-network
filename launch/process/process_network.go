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
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/util"
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

	if nt, err := NewNetworkServer(conf.Bind().Host, conf.URL(), encs); err != nil {
		return ctx, err
	} else {
		if i, ok := nt.(logging.SetLogger); ok {
			_ = i.SetLogger(l)
		}

		ctx = context.WithValue(ctx, ContextValueNetwork, nt)

		return ctx, nil
	}
}

func NewNetworkServer(bind string, u *url.URL, encs *encoder.Encoders) (network.Server, error) {
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
	} else if nqs, err := quicnetwork.NewServer(qs, encs, je); err != nil {
		return nil, err
	} else if err := nqs.Initialize(); err != nil {
		return nil, err
	} else {
		return nqs, nil
	}
}

func LoadNodeChannel(u *url.URL, encs *encoder.Encoders) (network.Channel, error) {
	var je encoder.Encoder
	if e, err := encs.Encoder(jsonenc.JSONType, ""); err != nil {
		return nil, xerrors.Errorf("json encoder needs for quic-network: %w", err)
	} else {
		je = e
	}

	switch u.Scheme {
	case "quic":
		if ch, err := quicnetwork.NewChannel(
			u.String(),
			100,
			true,
			time.Second*1000,
			3,
			nil,
			encs,
			je,
		); err != nil {
			return nil, err
		} else {
			return ch, nil
		}
	default:
		return nil, xerrors.Errorf("unsupported publish URL, %v", u.String())
	}
}
