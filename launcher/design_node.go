package launcher

import (
	"crypto/tls"
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

var DefaultNetworkBind = "0.0.0.0:54321"

type NodeDesign struct {
	encs             *encoder.Encoders
	jenc             *jsonenc.Encoder
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
	Component        *ComponentDesign
	Config           *NodeConfigDesign
}

func (nd *NodeDesign) Encoder() *encoder.Encoders {
	return nd.encs
}

func (nd *NodeDesign) JSONEncoder() *jsonenc.Encoder {
	return nd.jenc
}

func (nd *NodeDesign) SetEncoders(encs *encoder.Encoders) error {
	if e, err := encs.Encoder(jsonenc.JSONType, ""); err != nil {
		return xerrors.Errorf("json encoder needs for load design: %w", err)
	} else {
		nd.jenc = e.(*jsonenc.Encoder)
	}

	nd.encs = encs

	return nil
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
		nd.Component = NewComponentDesign(nil)
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

type BaseNetworkDesign struct {
	Bind        string
	Publish     string
	CertKeyFile string `yaml:"cert-key"`
	CertFile    string `yaml:"cert"`
	bindHost    string
	bindPort    int
	publishURL  *url.URL
	certs       []tls.Certificate
}

func (nd *BaseNetworkDesign) BindHost() string {
	return nd.bindHost
}

func (nd *BaseNetworkDesign) BindPort() int {
	return nd.bindPort
}

func (nd *BaseNetworkDesign) PublishURL() *url.URL {
	return nd.publishURL
}

func (nd *BaseNetworkDesign) Certs() []tls.Certificate {
	return nd.certs
}

func (nd *BaseNetworkDesign) IsValid([]byte) error {
	if nd == nil {
		return xerrors.Errorf("empty network design")
	}

	if len(nd.Bind) < 1 {
		nd.Bind = DefaultNetworkBind
	}

	if h, p, err := net.SplitHostPort(nd.Bind); err != nil {
		return xerrors.Errorf("invalid bind value, '%v': %w", nd.Bind, err)
	} else if i, err := strconv.ParseUint(p, 10, 64); err != nil {
		return xerrors.Errorf("invalid port in bind value, '%v': %w", nd.Bind, err)
	} else {
		nd.bindHost = h
		nd.bindPort = int(i)
	}

	if u, err := IsvalidNetworkURL(nd.Publish); err != nil {
		return err
	} else {
		nd.publishURL = u
	}

	if len(nd.CertKeyFile) > 0 && len(nd.CertFile) > 0 {
		if c, err := tls.LoadX509KeyPair(nd.CertFile, nd.CertKeyFile); err != nil {
			return err
		} else {
			nd.certs = []tls.Certificate{c}
		}
	}

	return nil
}

type NetworkDesign struct {
	sync.RWMutex `yaml:"-"`
	*BaseNetworkDesign
	publishURLWithIP *url.URL
}

func (nd *NetworkDesign) MarshalYAML() (interface{}, error) {
	return nd.BaseNetworkDesign, nil
}

func (nd *NetworkDesign) UnmarshalYAML(value *yaml.Node) error {
	var bn *BaseNetworkDesign
	if err := value.Decode(&bn); err != nil {
		return err
	} else {
		nd.BaseNetworkDesign = bn

		return nil
	}
}

func (nd *NetworkDesign) IsValid([]byte) error {
	if err := nd.BaseNetworkDesign.IsValid(nil); err != nil {
		return err
	}

	if u, err := IsvalidNodeNetworkURL(nd.Publish); err != nil {
		return err
	} else {
		nd.publishURL = u
	}

	return nil
}

func (nd *NetworkDesign) SetPublishURLWithIP(s string) (*NetworkDesign, error) {
	if u, err := IsvalidNodeNetworkURL(s); err != nil {
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

	if u, err := IsvalidNodeNetworkURL(rd.Network); err != nil {
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

func IsvalidNetworkURL(n string) (*url.URL, error) {
	if u, err := url.Parse(n); err != nil {
		return nil, xerrors.Errorf("invalid network url, '%v': %w", n, err)
	} else {
		return u, nil
	}
}

func IsvalidNodeNetworkURL(n string) (*url.URL, error) {
	if ur, err := IsvalidNetworkURL(n); err != nil {
		return nil, err
	} else {
		switch ur.Scheme {
		case "quic":
		default:
			return nil, xerrors.Errorf("unsupported network type found: %v", n)
		}

		return ur, nil
	}
}

func LoadNodeDesign(b []byte, encs *encoder.Encoders) (*NodeDesign, error) {
	var design *NodeDesign
	if err := yaml.Unmarshal(b, &design); err != nil {
		return nil, err
	}

	if err := design.SetEncoders(encs); err != nil {
		return nil, err
	}

	return design, nil
}

func LoadNodeDesignFromFile(f string, encs *encoder.Encoders) (*NodeDesign, error) {
	if b, err := ioutil.ReadFile(filepath.Clean(f)); err != nil {
		return nil, err
	} else {
		return LoadNodeDesign(b, encs)
	}
}
