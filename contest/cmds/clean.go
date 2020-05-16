package cmds

import (
	dockerClient "github.com/docker/docker/client"

	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/util/logging"
)

type CleanCommand struct {
	Stop bool `help:"just stop containers instead of cleaning"`
}

func (cmd *CleanCommand) Run(log logging.Logger, _ *[]func()) error {
	// create docker env
	var dc *dockerClient.Client
	if c, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv); err != nil {
		return err
	} else {
		dc = c
	}

	if cmd.Stop {
		return contestlib.StopContainers(dc, log)
	}

	return contestlib.CleanContainers(dc, log)
}
