package cmds

import (
	dockerClient "github.com/docker/docker/client"

	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/util/logging"
)

type CleanCommand struct {
	Stop bool `help:"just stop containers instead of cleaning"`
}

func (cmd *CleanCommand) Run(log logging.Logger) error {
	// create docker env
	var dc *dockerClient.Client
	if c, err := dockerClient.NewEnvClient(); err != nil {
		return err
	} else {
		dc = c
	}

	if cmd.Stop {
		log.Info().Msg("containers stopped")
		return contestlib.StopContainers(dc)
	}

	log.Info().Msg("containers cleaned")

	if err := contestlib.CleanContainers(dc, log); err != nil {
		return err
	}

	log.Info().Msg("containers pruned")
	if err := contestlib.ContainersPrune(dc); err != nil {
		return err
	}

	log.Info().Msg("volumes pruned")
	return contestlib.VolumesPrune(dc)
}
