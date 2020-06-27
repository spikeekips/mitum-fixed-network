package contestlib

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"time"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/network"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type Launcher struct {
	*logging.Logging
	*launcher.Launcher
	design *NodeDesign
}

func NewLauncherFromDesign(design *NodeDesign, version util.Version) (*Launcher, error) {
	nr := &Launcher{design: design}

	if ca, err := NewContestAddress(design.Address); err != nil {
		return nil, err
	} else if bn, err := launcher.NewLauncher(ca, design.Privatekey(), design.NetworkID(), version); err != nil {
		return nil, err
	} else {
		nr.Launcher = bn
	}

	nr.Logging = logging.NewLogging(func(c logging.Context) logging.Emitter {
		return c.Str("module", "contest-node-runner")
	})
	return nr, nil
}

func (nr *Launcher) SetLogger(l logging.Logger) logging.Logger {
	_ = nr.Launcher.SetLogger(l)
	_ = nr.Logging.SetLogger(l)

	return nr.Log()
}

func (nr *Launcher) Design() *NodeDesign {
	return nr.design
}

func (nr *Launcher) Initialize() error {
	if err := nr.attachStorage(); err != nil {
		return err
	}

	if err := nr.attachNetwork(); err != nil {
		return err
	}

	if err := nr.attachNodeChannel(); err != nil {
		return err
	}

	if err := nr.attachRemoteNodes(); err != nil {
		return err
	}

	if err := nr.attachSuffrage(); err != nil {
		return err
	}

	if err := nr.attachProposalProcessor(); err != nil {
		return err
	}

	return nr.Launcher.Initialize()
}

func (nr *Launcher) attachStorage() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "storage")
	})
	l.Debug().Msg("trying to attach")

	if st, err := LoadStorage(nr.design.Storage, nr.Encoders()); err != nil {
		return err
	} else {
		_ = nr.SetStorage(st)
	}

	l.Debug().Msg("attached")

	return nil
}

func (nr *Launcher) attachNetwork() error {
	// TODO move under launcher
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "network")
	})
	l.Debug().Msg("trying to attach")

	var je encoder.Encoder
	if e, err := nr.Encoders().Encoder(jsonenc.JSONType, ""); err != nil { // NOTE get latest json encoder
		return xerrors.Errorf("json encoder needs for quic-network: %w", err)
	} else {
		je = e
	}

	nd := nr.design.Network

	var certs []tls.Certificate
	if priv, err := util.GenerateED25519Privatekey(); err != nil {
		return err
	} else if ct, err := util.GenerateTLSCerts(nd.PublishURL().Host, priv); err != nil {
		return err
	} else {
		certs = ct
	}

	if qs, err := quicnetwork.NewPrimitiveQuicServer(nd.Bind, certs); err != nil {
		return err
	} else if nqs, err := quicnetwork.NewQuicServer(qs, nr.Encoders(), je); err != nil {
		return err
	} else {
		_ = nr.SetNetwork(nqs)
	}

	_ = nr.SetPublichURL(nd.PublishURL().String())

	l.Debug().Msg("attached")

	return nil
}

func (nr *Launcher) attachNodeChannel() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "node-channel")
	})
	l.Debug().Msg("trying to attach")

	var je encoder.Encoder
	if e, err := nr.Encoders().Encoder(jsonenc.JSONType, ""); err != nil { // NOTE get latest json encoder
		return xerrors.Errorf("json encoder needs for quic-network: %w", err)
	} else {
		je = e
	}

	nu := new(url.URL)
	*nu = *nr.design.Network.PublishURL()
	nu.Host = fmt.Sprintf("localhost:%s", nu.Port())

	if ch, err := CreateNodeChannel(nu, nr.Encoders(), je); err != nil {
		return err
	} else {
		_ = nr.SetNodeChannel(ch)
		_ = nr.Localstate().Node().SetChannel(ch)
	}

	l.Debug().Msg("attached")

	return nil
}

func (nr *Launcher) attachRemoteNodes() error {
	var je encoder.Encoder
	if e, err := nr.Encoders().Encoder(jsonenc.JSONType, ""); err != nil { // NOTE get latest json encoder
		return xerrors.Errorf("json encoder needs for quic-network: %w", err)
	} else {
		je = e
	}

	nodes := make([]network.Node, len(nr.design.Nodes))

	for i, r := range nr.design.Nodes {
		r := r
		l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
			return ctx.Str("address", r.Address)
		})

		l.Debug().Msg("trying to create remote node")

		var n *isaac.RemoteNode
		if ca, err := NewContestAddress(r.Address); err != nil {
			return err
		} else {
			n = isaac.NewRemoteNode(ca, r.Publickey())
		}

		if ch, err := CreateNodeChannel(r.NetworkURL(), nr.Encoders(), je); err != nil {
			return err
		} else {
			_ = n.SetChannel(ch)
		}
		l.Debug().Msg("created")

		nodes[i] = n
	}

	return nr.Localstate().Nodes().Add(nodes...)
}

func (nr *Launcher) attachSuffrage() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "suffrage")
	})
	l.Debug().Msg("trying to attach")

	if sf, err := nr.design.Component.Suffrage.New(nr.Localstate()); err != nil {
		return xerrors.Errorf("failed to create new suffrage component: %w", err)
	} else {
		l.Debug().
			Str("type", nr.design.Component.Suffrage.Type).
			Interface("info", nr.design.Component.Suffrage.Info).
			Msg("suffrage loaded")

		_ = nr.SetSuffrage(sf)
	}

	l.Debug().Msg("attached")

	return nil
}

func (nr *Launcher) attachProposalProcessor() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "proposal-processor")
	})
	l.Debug().Msg("trying to attach")

	if pp, err := nr.design.Component.ProposalProcessor.New(nr.Localstate(), nr.Suffrage()); err != nil {
		return xerrors.Errorf("failed to create new proposal processor component: %w", err)
	} else {
		l.Debug().
			Str("type", nr.design.Component.ProposalProcessor.Type).
			Interface("info", nr.design.Component.ProposalProcessor.Info).
			Msg("proposal processor loaded")

		_ = nr.SetProposalProcessor(pp)
	}

	l.Debug().Msg("attached")

	return nil
}

func CreateNodeChannel(publish *url.URL, encs *encoder.Encoders, enc encoder.Encoder) (network.NetworkChannel, error) {
	var channel network.NetworkChannel

	switch publish.Scheme {
	case "quic":
		if ch, err := quicnetwork.NewQuicChannel(
			publish.String(),
			100,
			true,
			time.Second*1,
			3,
			nil,
			encs,
			enc,
		); err != nil {
			return nil, err
		} else {
			channel = ch
		}
	default:
		return nil, xerrors.Errorf("unsupported publish URL, %v", publish.String())
	}

	return channel, nil
}
