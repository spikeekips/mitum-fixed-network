package cmds

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/launch"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
)

type StartHandoverCommand struct {
	*BaseCommand
	Address    string        `arg:"" name:"node address" required:"true"`
	Key        string        `arg:"" name:"private key of node" required:"true"`
	NetworkID  string        `arg:"" name:"network-id" required:"true"`
	URL        *url.URL      `arg:"" name:"new node url" help:"new node url" required:"true"`
	Timeout    time.Duration `name:"timeout" help:"timeout; default is 5 seconds"`
	TLSInscure bool          `name:"tls-insecure" help:"allow inseucre TLS connection; default is false"`
	address    base.Address
	privatekey key.Privatekey
	networkID  base.NetworkID
}

func NewStartHandoverCommand() StartHandoverCommand {
	return StartHandoverCommand{
		BaseCommand: NewBaseCommand("start_handover"),
	}
}

func (cmd *StartHandoverCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
	}

	encs := cmd.Encoders()
	if encs == nil {
		i, err := cmd.LoadEncoders(launch.EncoderTypes, launch.EncoderHinters)
		if err != nil {
			return err
		}
		encs = i
	}

	switch i, err := base.DecodeAddressFromString(cmd.Address, cmd.jsonenc); {
	case err != nil:
		return errors.Wrap(err, "failed to load address")
	default:
		cmd.address = i
	}

	if i, err := loadKey([]byte(cmd.Key), cmd.jsonenc); err != nil {
		return errors.Wrap(err, "failed to load privatekey")
	} else if j, ok := i.(key.Privatekey); !ok {
		return errors.Errorf("failed to load privatekey; not privatekey, %T", i)
	} else {
		cmd.privatekey = j

		cmd.Log().Debug().Stringer("privatekey", cmd.privatekey).Msg("privatekey loaded")
	}

	cmd.networkID = base.NetworkID([]byte(cmd.NetworkID))
	cmd.Log().Debug().Str("network_id", cmd.NetworkID).Msg("network-id loaded")

	if cmd.Timeout < 1 {
		cmd.Timeout = time.Second * 5
	}

	connInfo := network.NewHTTPConnInfo(network.NormalizeURL(cmd.URL), cmd.TLSInscure)
	channel, err := process.LoadNodeChannel(connInfo, encs, cmd.Timeout)
	if err != nil {
		return err
	}
	cmd.Log().Debug().Msg("network channel loaded")

	sl, err := network.NewHandoverSealV0(
		network.StartHandoverSealV0Hint,
		cmd.privatekey,
		cmd.address,
		connInfo,
		cmd.networkID,
	)
	if err != nil {
		return fmt.Errorf("failed to make StartHandoverSealV: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cmd.Timeout)
	defer cancel()

	switch ok, err := channel.StartHandover(ctx, sl); {
	case err != nil:
		return fmt.Errorf("failed to start handover: %w", err)
	case !ok:
		return errors.Errorf("failed to start handover")
	default:
		cmd.Log().Info().Msg("handover started")

		return nil
	}
}
