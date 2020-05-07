package contestlib

import (
	"io/ioutil"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/spikeekips/mitum/util/isvalid"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v2"
)

type NodeDesign struct {
	Address         string `yaml:"address"`
	Storage         string `yaml:"storage"`
	NetworkIDString string `yaml:"network-id"`
	Network         *NetworkDesign
}

func LoadDesignFromFile(f string) (*NodeDesign, error) {
	var design NodeDesign
	if b, err := ioutil.ReadFile(f); err != nil {
		return nil, err
	} else if err := yaml.Unmarshal([]byte(b), &design); err != nil {
		return nil, err
	}

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

	return nil
}

func (nd *NodeDesign) NetworkID() []byte {
	return []byte(nd.NetworkIDString)
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

func (nd *NetworkDesign) BindHost() string {
	return nd.bindHost
}

func (nd *NetworkDesign) BindPort() int {
	return nd.bindPort
}

func (nd *NetworkDesign) PublishURL() *url.URL {
	return nd.publishURL
}
