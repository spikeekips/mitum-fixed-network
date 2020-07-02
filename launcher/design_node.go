package launcher

import (
	"net"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/isvalid"
)

type NodeDesign struct {
	encs             *encoder.Encoders
	Address          string
	PrivatekeyString string `yaml:"privatekey"`
	Storage          string
	NetworkIDString  string `yaml:"network-id,omitempty"`
	Network          *NetworkDesign
	GenesisPolicy    *PolicyDesign `yaml:"genesis-policy,omitempty"`
	privatekey       key.Privatekey
	Nodes            []*RemoteDesign
}

func (nd *NodeDesign) SetEncoders(encs *encoder.Encoders) {
	nd.encs = encs
}

func (nd *NodeDesign) IsValid([]byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		nd.Network,
	}, nil, true); err != nil {
		return err
	}

	if len(strings.TrimSpace(nd.NetworkIDString)) < 1 {
		nd.NetworkIDString = "contest-network-id"
	}

	var je encoder.Encoder
	if e, err := nd.encs.Encoder(jsonenc.JSONType, ""); err != nil {
		return xerrors.Errorf("json encoder needs for load design: %w", err)
	} else {
		je = e
	}

	if pk, err := key.DecodePrivatekey(je, nd.PrivatekeyString); err != nil {
		return err
	} else {
		nd.privatekey = pk
	}

	addrs := map[string]struct{}{
		nd.Address: {},
	}
	for _, r := range nd.Nodes {
		r.encs = nd.encs
		if err := r.IsValid(nil); err != nil {
			return err
		}

		if _, found := addrs[r.Address]; found {
			return xerrors.Errorf("duplicated address found: '%v'", r.Address)
		}
		addrs[r.Address] = struct{}{}
	}

	if nd.GenesisPolicy == nil {
		nd.GenesisPolicy = NewPolicyDesign()
	}
	if err := nd.GenesisPolicy.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (nd NodeDesign) NetworkID() []byte {
	return []byte(nd.NetworkIDString)
}

func (nd NodeDesign) Privatekey() key.Privatekey {
	return nd.privatekey
}

type NetworkDesign struct {
	Bind       string
	Publish    string
	bindHost   string
	bindPort   int
	publishURL *url.URL
}

func (nd *NetworkDesign) IsValid([]byte) error {
	if nd == nil {
		return xerrors.Errorf("empty network design")
	}

	if h, p, err := net.SplitHostPort(nd.Bind); err != nil {
		return xerrors.Errorf("invalid bind value, '%v': %w", nd.Bind, err)
	} else if i, err := strconv.ParseUint(p, 10, 64); err != nil {
		return xerrors.Errorf("invalid port in bind value, '%v': %w", nd.Bind, err)
	} else {
		nd.bindHost = h
		nd.bindPort = int(i)
	}

	if u, err := isvalidNetworkURL(nd.Publish); err != nil {
		return err
	} else {
		nd.publishURL = u
	}

	return nil
}

func (nd *NetworkDesign) PublishURL() *url.URL {
	return nd.publishURL
}

type RemoteDesign struct {
	encs            *encoder.Encoders
	Address         string
	PublickeyString string `yaml:"publickey"`
	Network         string
	publickey       key.Publickey
	networkURL      *url.URL
}

func (rd *RemoteDesign) IsValid([]byte) error {
	var je encoder.Encoder
	if e, err := rd.encs.Encoder(jsonenc.JSONType, ""); err != nil {
		return xerrors.Errorf("json encoder needs for load design: %w", err)
	} else {
		je = e
	}

	if pk, err := key.DecodePublickey(je, rd.PublickeyString); err != nil {
		return err
	} else {
		rd.publickey = pk
	}

	if u, err := isvalidNetworkURL(rd.Network); err != nil {
		return err
	} else {
		rd.networkURL = u
	}

	return nil
}

func (rd *RemoteDesign) Publickey() key.Publickey {
	return rd.publickey
}

func (rd *RemoteDesign) NetworkURL() *url.URL {
	return rd.networkURL
}

func isvalidNetworkURL(n string) (*url.URL, error) {
	var ur *url.URL
	if u, err := url.Parse(n); err != nil {
		return nil, xerrors.Errorf("invalid network url, '%v': %w", n, err)
	} else {
		ur = u
	}

	switch ur.Scheme {
	case "quic":
	default:
		return nil, xerrors.Errorf("unsupported network type found: %v", n)
	}

	return ur, nil
}
