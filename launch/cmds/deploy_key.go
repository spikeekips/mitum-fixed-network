package cmds

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/lucas-clemente/quic-go"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/launch/deploy"
	"github.com/spikeekips/mitum/network"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"golang.org/x/xerrors"
)

var NodeConnectVars = kong.Vars{
	"node_connect_timeout":      "5s",
	"node_connect_tls_insecure": "false",
}

type NodeConnectFlags struct {
	URL        *url.URL      `arg:"" name:"node url" help:"remote mitum url; default: ${node_url}" required:"true" default:"${node_url}"`                             // revive:disable-line:line-length-limit
	Timeout    time.Duration `name:"timeout" help:"timeout; default ${node_connect_timeout}" default:"${node_connect_timeout}"`                                       // revive:disable-line:line-length-limit
	TLSInscure bool          `name:"tls-insecure" help:"allow inseucre TLS connection; default: ${node_connect_tls_insecure}" default:"${node_connect_tls_insecure}"` // revive:disable-line:line-length-limit,struct-tag
}

type baseDeployKeyCommand struct {
	*BaseCommand
	Key       string `arg:"" name:"private key of node" required:"true"`
	NetworkID string `arg:"" name:"network-id" required:"true"`
	*NodeConnectFlags
	privatekey key.Privatekey
	networkID  base.NetworkID
	client     *quicnetwork.QuicClient
	token      string
}

func newBaseDeployKeyCommand(name string) *baseDeployKeyCommand {
	return &baseDeployKeyCommand{
		BaseCommand:      NewBaseCommand(name),
		NodeConnectFlags: &NodeConnectFlags{},
	}
}

func (cmd *baseDeployKeyCommand) Initialize(flags interface{}, version util.Version) error {
	cmd.BaseCommand.LogOutput = os.Stderr

	if err := cmd.BaseCommand.Initialize(flags, version); err != nil {
		return err
	} else if _, err := cmd.LoadEncoders(nil, nil); err != nil {
		return err
	}

	if i, err := loadKey(cmd.jsonenc, []byte(cmd.Key)); err != nil {
		return xerrors.Errorf("failed to load node privatekey: %w", err)
	} else if j, ok := i.(key.Privatekey); !ok {
		return xerrors.Errorf("failed to load node privatekey; not privatekey, %T", i)
	} else {
		cmd.privatekey = j

		cmd.Log().Debug().Str("node_privatekey", cmd.privatekey.String()).Msg("node privatekey loaded")
	}

	cmd.networkID = base.NetworkID([]byte(cmd.NetworkID))

	cmd.Log().Debug().Str("network_id", cmd.NetworkID).Msg("network-id loaded")

	if cmd.Timeout < 1 {
		cmd.Timeout = time.Second * 5
	}

	if cmd.URL.Scheme == "quic" {
		cmd.URL.Scheme = "https"
	}

	cmd.Log().Debug().Interface("node_url", cmd.URL).Msg("deploy key")

	quicConfig := &quic.Config{HandshakeIdleTimeout: cmd.Timeout}
	i, err := quicnetwork.NewQuicClient(cmd.TLSInscure, quicConfig)
	if err != nil {
		return err
	}
	cmd.client = i

	return nil
}

func (cmd *baseDeployKeyCommand) requestToken() error {
	ctx, cancel := context.WithTimeout(context.Background(), cmd.Timeout)
	defer cancel()

	var body []byte

	u := *cmd.URL
	u.Path = filepath.Join(u.Path, deploy.QuicHandlerPathDeployKeyToken)
	if i, err := cmd.client.Get(ctx, cmd.Timeout, u.String(), nil, nil); err != nil {
		return err
	} else if i.StatusCode != http.StatusOK {
		if j, err := network.LoadProblemFromResponse(i.Response); err == nil {
			cmd.Log().Debug().Interface("response", i).Interface("problem", j).Msg("failed to request token")

			return j
		}

		cmd.Log().Debug().Interface("response", i).Msg("failed to request token")

		return xerrors.Errorf("failed to request token")
	} else if j, err := ioutil.ReadAll(i.Body()); err != nil {
		return xerrors.Errorf("failed to read body for requesting token")
	} else {
		body = j
	}

	var m map[string]string

	if err := jsonenc.Unmarshal(body, &m); err != nil {
		return xerrors.Errorf("failed to load token response: %w", err)
	} else if i, found := m["token"]; !found {
		return xerrors.Errorf("token not found in response")
	} else {
		cmd.token = i
	}

	cmd.Log().Debug().Str("token", cmd.token).Msg("token received")

	return nil
}

func (cmd *baseDeployKeyCommand) requestWithToken(path, method string) (*http.Response, func() error, error) {
	sig, err := deploy.DeployKeyTokenSignature(cmd.privatekey, cmd.token, cmd.networkID)
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to make signature with token: %w", err)
	}

	u := *cmd.URL
	u.Path = filepath.Join(u.Path, path)
	query := u.Query()
	query.Add("token", cmd.token)
	query.Add("signature", sig.String())

	u.RawQuery = query.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), cmd.Timeout)
	defer cancel()

	return cmd.client.Request(ctx, cmd.Timeout, u.String(), method, nil, nil)
}

func loadKey(enc encoder.Encoder, b []byte) (key.Key, error) {
	s := strings.TrimSpace(string(b))

	if pk, err := key.DecodeKey(enc, s); err != nil {
		return nil, err
	} else if err := pk.IsValid(nil); err != nil {
		return nil, err
	} else {
		return pk, nil
	}
}
