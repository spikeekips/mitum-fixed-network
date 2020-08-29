package launcher

import (
	"io/ioutil"
	"net"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/isvalid"
)

type NodeDesign struct {
	encs             *encoder.Encoders
	address          base.Address
	AddressString    string `yaml:"address"`
	PrivatekeyString string `yaml:"privatekey"`
	BlockFS          string `yaml:"blockfs"`
	Storage          string
	NetworkIDString  string `yaml:"network-id,omitempty"`
	Network          *NetworkDesign
	GenesisPolicy    *PolicyDesign `yaml:"genesis-policy,omitempty"`
	privatekey       key.Privatekey
	Nodes            []*RemoteDesign
	InitOperations   []OperationDesign `yaml:"init-operations"`
	Component        *NodeComponentDesign
	Config           *NodeConfigDesign
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

	if len(strings.TrimSpace(nd.BlockFS)) < 1 {
		return xerrors.Errorf("blockfs must be given")
	}

	var je encoder.Encoder
	if e, err := nd.encs.Encoder(jsonenc.JSONType, ""); err != nil {
		return xerrors.Errorf("json encoder needs for load design: %w", err)
	} else {
		je = e
	}

	if err := nd.isValidAddress(je); err != nil {
		return err
	}

	if err := nd.isValidPrivatekey(je); err != nil {
		return err
	}

	if err := nd.isValidRemotes(); err != nil {
		return err
	}

	if nd.GenesisPolicy != nil {
		if err := nd.GenesisPolicy.IsValid(nil); err != nil {
			return err
		}
	}

	if nd.Component == nil {
		nd.Component = NewNodeComponentDesign(nil)
	}
	if err := nd.Component.IsValid(nil); err != nil {
		return err
	}

	if nd.Config == nil {
		nd.Config = NewNodeConfigDesign(nil)
	}
	if err := nd.Config.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (nd *NodeDesign) isValidAddress(je encoder.Encoder) error {
	if a, err := base.DecodeAddressFromString(je, strings.TrimSpace(nd.AddressString)); err != nil {
		return err
	} else if err := a.IsValid(nil); err != nil {
		return err
	} else {
		nd.address = a
	}

	return nil
}

func (nd *NodeDesign) isValidPrivatekey(je encoder.Encoder) error {
	if pk, err := key.DecodePrivatekey(je, nd.PrivatekeyString); err != nil {
		return err
	} else {
		nd.privatekey = pk
	}

	return nil
}

func (nd *NodeDesign) isValidRemotes() error {
	addrs := map[string]struct{}{
		nd.Address().String(): {},
	}
	for _, r := range nd.Nodes {
		r.encs = nd.encs
		if err := r.IsValid(nil); err != nil {
			return err
		}

		if _, found := addrs[r.Address().String()]; found {
			return xerrors.Errorf("duplicated address found: '%v'", r.Address().String())
		}
		addrs[r.Address().String()] = struct{}{}
	}

	return nil
}

func (nd NodeDesign) Address() base.Address {
	return nd.address
}

func (nd NodeDesign) NetworkID() []byte {
	return []byte(nd.NetworkIDString)
}

func (nd NodeDesign) Privatekey() key.Privatekey {
	return nd.privatekey
}

type NetworkDesign struct {
	sync.RWMutex
	Bind             string
	Publish          string
	bindHost         string
	bindPort         int
	publishURL       *url.URL
	publishURLWithIP *url.URL
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

func (nd *NetworkDesign) SetPublishURLWithIP(s string) (*NetworkDesign, error) {
	if u, err := isvalidNetworkURL(s); err != nil {
		return nil, err
	} else {
		nd.Lock()
		defer nd.Unlock()

		nd.publishURLWithIP = u
	}

	return nd, nil
}

func (nd *NetworkDesign) PublishURLWithIP() *url.URL {
	nd.RLock()
	defer nd.RUnlock()

	return nd.publishURLWithIP
}

type RemoteDesign struct {
	encs            *encoder.Encoders
	address         base.Address
	AddressString   string `yaml:"address"`
	PublickeyString string `yaml:"publickey"`
	Network         string
	publickey       key.Publickey
	networkURL      *url.URL
}

func (rd *RemoteDesign) SetEncoders(encs *encoder.Encoders) {
	rd.encs = encs
}

func (rd *RemoteDesign) IsValid([]byte) error {
	var je encoder.Encoder
	if e, err := rd.encs.Encoder(jsonenc.JSONType, ""); err != nil {
		return xerrors.Errorf("json encoder needs for load design: %w", err)
	} else {
		je = e
	}

	if a, err := base.DecodeAddressFromString(je, strings.TrimSpace(rd.AddressString)); err != nil {
		return err
	} else if err := a.IsValid(nil); err != nil {
		return err
	} else {
		rd.address = a
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

func (rd *RemoteDesign) Address() base.Address {
	return rd.address
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

func LoadNodeDesign(b []byte, encs *encoder.Encoders) (*NodeDesign, error) {
	var design *NodeDesign
	if err := yaml.Unmarshal(b, &design); err != nil {
		return nil, err
	}

	design.SetEncoders(encs)

	return design, nil
}

func LoadNodeDesignFromFile(f string, encs *encoder.Encoders) (*NodeDesign, error) {
	if b, err := ioutil.ReadFile(filepath.Clean(f)); err != nil {
		return nil, err
	} else {
		return LoadNodeDesign(b, encs)
	}
}
