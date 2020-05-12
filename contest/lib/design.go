package contestlib

import (
	"io/ioutil"
	"net"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v2"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/isvalid"
)

type NodeDesign struct {
	encs             *encoder.Encoders
	Address          string
	PrivatekeyString string `yaml:"privatekey"`
	Storage          string
	NetworkIDString  string `yaml:"network-id"`
	Network          *NetworkDesign
	privatekey       key.Privatekey
	Nodes            []*RemoteDesign
}

func LoadDesignFromFile(f string, encs *encoder.Encoders) (*NodeDesign, error) {
	var design NodeDesign
	if b, err := ioutil.ReadFile(f); err != nil {
		return nil, err
	} else if err := yaml.Unmarshal([]byte(b), &design); err != nil {
		return nil, err
	}

	design.encs = encs

	return &design, nil
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
	if e, err := nd.encs.Encoder(jsonencoder.JSONType, ""); err != nil {
		return xerrors.Errorf("json encoder needs for load design", err)
	} else {
		je = e
	}

	if pk, err := key.DecodePrivatekey(je, []byte(nd.PrivatekeyString)); err != nil {
		return err
	} else {
		nd.privatekey = pk
	}

	addrs := map[string]struct{}{
		nd.Address: struct{}{},
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

	return nil
}

func (nd *NodeDesign) NetworkID() []byte {
	return []byte(nd.NetworkIDString)
}

func (nd *NodeDesign) Privatekey() key.Privatekey {
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
	if h, p, err := net.SplitHostPort(nd.Bind); err != nil {
		return xerrors.Errorf("invalid bind value, '%v': %w", nd.Bind, err)
	} else if i, err := strconv.ParseUint(p, 10, 64); err != nil {
		return xerrors.Errorf("invalid port in bind value, '%v': %w", nd.Bind, err)
	} else {
		nd.bindHost = h
		nd.bindPort = int(i)
	}

	if u, err := url.Parse(nd.Publish); err != nil {
		return xerrors.Errorf("invalid publish url, '%v': %w", nd.Publish, err)
	} else {
		nd.publishURL = u
	}

	switch nd.publishURL.Scheme {
	case "quic":
	default:
		return xerrors.Errorf("unsupported network type found: %v", nd.Publish)
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
	if e, err := rd.encs.Encoder(jsonencoder.JSONType, ""); err != nil {
		return xerrors.Errorf("json encoder needs for load design", err)
	} else {
		je = e
	}

	if pk, err := key.DecodePublickey(je, []byte(rd.PublickeyString)); err != nil {
		return err
	} else {
		rd.publickey = pk
	}

	if u, err := url.Parse(rd.Network); err != nil {
		return xerrors.Errorf("invalid network url, '%v': %w", rd.Network, err)
	} else {
		rd.networkURL = u
	}

	switch rd.networkURL.Scheme {
	case "quic":
	default:
		return xerrors.Errorf("unsupported network type found: %v", rd.networkURL)
	}

	return nil
}

func (rd *RemoteDesign) Publickey() key.Publickey {
	return rd.publickey
}

func (rd *RemoteDesign) NetworkURL() *url.URL {
	return rd.networkURL
}
