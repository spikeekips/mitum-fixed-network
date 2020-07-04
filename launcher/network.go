package launcher

import (
	"crypto/tls"
	"net/url"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/network"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func LoadNetworkServer(bind string, u *url.URL, encs *encoder.Encoders) (network.Server, error) {
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
	} else if nqs, err := quicnetwork.NewQuicServer(qs, encs, je); err != nil {
		return nil, err
	} else {
		return nqs, nil
	}
}

func LoadNodeChannel(u *url.URL, encs *encoder.Encoders) (network.NetworkChannel, error) {
	var je encoder.Encoder
	if e, err := encs.Encoder(jsonenc.JSONType, ""); err != nil {
		return nil, xerrors.Errorf("json encoder needs for quic-network: %w", err)
	} else {
		je = e
	}

	switch u.Scheme {
	case "quic":
		if ch, err := quicnetwork.NewQuicChannel(
			u.String(),
			100,
			true,
			time.Second*1,
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
