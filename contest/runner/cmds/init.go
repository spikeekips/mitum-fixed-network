package cmds

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/policy"
	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type InitCommand struct {
	Design  string `arg:"" name:"node design file" help:"node design file" type:"existingfile"`
	Force   bool   `help:"clean the existing environment"`
	version util.Version
}

func (cmd *InitCommand) Run(log logging.Logger, version util.Version) error {
	log.Info().Msg("trying to initialize")

	cmd.version = version

	return cmd.run(log)
}

func (cmd *InitCommand) run(log logging.Logger) error {
	var nr *contestlib.Launcher
	if n, err := createLauncherFromDesign(cmd.Design, cmd.version, log); err != nil {
		return err
	} else {
		nr = n
	}

	var ops []operation.Operation
	if op, err := cmd.loadPolicyOperation(nr.Design()); err != nil {
		return err
	} else {
		ops = append(ops, op)
	}

	log.Debug().Int("operations", len(ops)).Msg("operations loaded")

	if err := nr.Initialize(); err != nil {
		return xerrors.Errorf("failed to generate node from design: %w", err)
	}

	log.Debug().Msg("checking existing blocks")

	if err := cmd.checkExisting(nr, log); err != nil {
		return err
	}

	log.Debug().Msg("trying to create genesis block")
	if gg, err := isaac.NewGenesisBlockV0Generator(nr.Localstate(), ops); err != nil {
		return xerrors.Errorf("failed to create genesis block generator: %w", err)
	} else if blk, err := gg.Generate(); err != nil {
		return xerrors.Errorf("failed to generate genesis block: %w", err)
	} else {
		log.Info().
			Dict("block", logging.Dict().Hinted("height", blk.Height()).Hinted("hash", blk.Hash())).
			Msg("genesis block created")
	}

	log.Info().Msg("genesis block created")
	log.Info().Msg("iniialized")

	return nil
}

func (cmd *InitCommand) checkExisting(nr *contestlib.Launcher, log logging.Logger) error {
	log.Debug().Msg("checking existing blocks")

	var manifest block.Manifest
	if m, found, err := nr.Storage().LastManifest(); err != nil {
		return err
	} else if found {
		manifest = m
	}

	if manifest == nil {
		log.Debug().Msg("not found existing blocks")
	} else {
		log.Debug().Msgf("found existing blocks: block=%d", manifest.Height())

		if !cmd.Force {
			return xerrors.Errorf("environment already exists: block=%d", manifest.Height())
		}

		if err := nr.Storage().Clean(); err != nil {
			return err
		}
		log.Debug().Msg("existing environment cleaned")
	}

	return nil
}

func (cmd *InitCommand) loadPolicyOperation(design *launcher.NodeDesign) (
	operation.Operation, error,
) {
	token := []byte("genesis-policies-from-contest")

	if op, err := policy.NewSetPolicyV0(
		design.GenesisPolicy.Policy().(policy.PolicyV0),
		token,
		design.Privatekey(),
		design.NetworkID(),
	); err != nil {
		return nil, xerrors.Errorf("failed to create SetPolicyOperation: %w", err)
	} else {
		return op, nil
	}
}
