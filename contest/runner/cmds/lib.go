package cmds

import (
	"net/url"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
)

func createLauncherFromDesign(f string, version util.Version, log logging.Logger) (*contestlib.Launcher, error) {
	var encs *encoder.Encoders
	if e, err := encoder.LoadEncoders(
		[]encoder.Encoder{jsonenc.NewEncoder(), bsonenc.NewEncoder()},
		contestlib.Hinters...,
	); err != nil {
		return nil, xerrors.Errorf("failed to load encoders: %w", err)
	} else {
		encs = e
	}

	var design *contestlib.NodeDesign
	if d, err := loadDesign(f, encs); err != nil {
		return nil, xerrors.Errorf("failed to load design: %w", err)
	} else {
		design = d
	}

	var nr *contestlib.Launcher
	if n, err := contestlib.NewLauncherFromDesign(design, version); err != nil {
		return nil, xerrors.Errorf("failed to create node runner: %w", err)
	} else if err := n.AddHinters(contestlib.Hinters...); err != nil {
		return nil, err
	} else {
		nr = n
	}

	log.Debug().Interface("design", design).Msg("load launcher from design")
	_ = nr.SetLogger(log)

	return nr, nil
}

func loadDesign(f string, encs *encoder.Encoders) (*contestlib.NodeDesign, error) {
	if d, err := contestlib.LoadNodeDesignFromFile(f, encs); err != nil {
		return nil, xerrors.Errorf("failed to load design file: %w", err)
	} else if err := d.IsValid(nil); err != nil {
		return nil, xerrors.Errorf("invalid design file: %w", err)
	} else {
		return d, nil
	}
}

func loadNodeChannel(u *url.URL, log logging.Logger) (network.NetworkChannel, error) {
	var encs *encoder.Encoders
	if e, err := encoder.LoadEncoders(
		[]encoder.Encoder{jsonenc.NewEncoder(), bsonenc.NewEncoder()},
		contestlib.Hinters...,
	); err != nil {
		return nil, xerrors.Errorf("failed to load encoders: %w", err)
	} else {
		encs = e
	}
	log.Debug().Msg("hinters loaded")

	var channel network.NetworkChannel
	if ch, err := launcher.LoadNodeChannel(u, encs); err != nil {
		return nil, err
	} else {
		channel = ch
		if l, ok := channel.(logging.SetLogger); ok {
			_ = l.SetLogger(log)
		}
	}

	return channel, nil
}

func loadHeights(hs []int64) ([]base.Height, error) {
	var heights []base.Height // nolint
	for _, i := range hs {
		h := base.Height(i)
		if err := h.IsValid(nil); err != nil {
			return nil, err
		}

		var found bool
		for _, m := range heights {
			if h == m {
				found = true
				break
			}
		}
		if found {
			continue
		}

		heights = append(heights, h)
	}

	return heights, nil
}

func requestByHeights(u *url.URL, hs []int64, t string, log logging.Logger) (interface{}, error) {
	var heights []base.Height
	switch hs, err := loadHeights(hs); {
	case err != nil:
		return nil, err
	case len(hs) < 1:
		return nil, xerrors.Errorf("missing height")
	default:
		heights = hs
	}

	var channel network.NetworkChannel
	if ch, err := loadNodeChannel(u, log); err != nil {
		return nil, err
	} else {
		channel = ch
	}
	log.Debug().Msg("network channel loaded")

	switch t {
	case "blocks":
		return channel.Blocks(heights)
	case "manifests":
		return channel.Manifests(heights)
	default:
		return nil, xerrors.Errorf("unknown request: %s", t)
	}
}
