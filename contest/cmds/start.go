package cmds

import (
	"os"
	"path/filepath"
	"time"

	dockerClient "github.com/docker/docker/client"
	"golang.org/x/xerrors"

	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/logging"
)

var networkName = "contest-network"

type StartCommand struct {
	Image      string `help:"docker image for node runner (default: ${start_image})" default:"${start_image}"`
	Design     string `arg:"" name:"node design file" help:"contest design file" type:"existingfile"`
	RunnerPath string `arg:"" name:"runner-path" help:"mitum node runner, 'mitum-runner' path" type:"existingfile"`
	NotClean   bool   `help:"don't clean containers (default: ${start_not_clean})" default:"${start_not_clean}"`
	Output     string `help:"output directory" type:"existingdir"`
	log        logging.Logger
	exitHooks  *[]func()
	design     *contestlib.ContestDesign
	encs       *encoder.Encoders
}

func (cmd *StartCommand) Run(log logging.Logger, exitHooks *[]func()) error {
	cmd.log = log
	cmd.exitHooks = exitHooks

	if e, err := contestlib.LoadEncoder(); err != nil {
		return xerrors.Errorf("failed to load encoders: %w", err)
	} else {
		cmd.encs = e
	}

	// load design file
	if err := cmd.loadDesign(cmd.Design); err != nil {
		return err
	}

	// create docker env
	var dc *dockerClient.Client
	if c, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv); err != nil {
		return err
	} else {
		dc = c
	}

	cmd.log.Debug().Strs("image", []string{cmd.Image, contestlib.MongodbImage}).Msg("trying to pull image")
	if err := contestlib.PullImages(dc, cmd.Image, contestlib.MongodbImage); err != nil {
		return err
	}

	cmd.log.Debug().Msg("trying to create docker network")

	if err := contestlib.CleanContainers(dc, cmd.log); err != nil {
		return err
	}

	cmd.log.Debug().Msg("trying to create containers")
	var cts *contestlib.Containers
	if c, err := cmd.createContainers(dc); err != nil {
		return err
	} else {
		cts = c
		cmd.log.Debug().Msg("containers created")
	}

	cmd.log.Debug().Msg("trying to run containers")

	return cts.Run()
}

func (cmd *StartCommand) createContainers(dc *dockerClient.Client) (*contestlib.Containers, error) {
	var dockerNetworkID string
	if i, err := contestlib.CreateDockerNetwork(dc, networkName, false); err != nil {
		return nil, xerrors.Errorf("failed to create new docker network: %w", err)
	} else {
		dockerNetworkID = i
	}

	output := filepath.Join(cmd.Output, Version, time.Now().Format("2006-01-02T15-04-05"))
	if err := os.MkdirAll(output, 0700); err != nil {
		return nil, err
	}

	var cts *contestlib.Containers
	if c, err := contestlib.NewContainers(
		dc,
		cmd.encs,
		cmd.Image,
		cmd.RunnerPath,
		networkName,
		dockerNetworkID,
		cmd.design,
		output,
	); err != nil {
		return nil, err
	} else {
		cts = c
		_ = cts.SetLogger(cmd.log)

		if !cmd.NotClean {
			contestlib.AddExitHook(cmd.exitHooks, func() {
				_ = contestlib.CleanContainers(dc, cmd.log)
				_ = cts.Clean()
			})
		}
	}

	return cts, nil
}

func (cmd *StartCommand) loadDesign(f string) error {
	if d, err := contestlib.LoadContestDesignFromFile(f, cmd.encs); err != nil {
		return xerrors.Errorf("failed to load design file: %w", err)
	} else if err := d.IsValid(nil); err != nil {
		return xerrors.Errorf("invalid design file: %w", err)
	} else {
		cmd.log.Debug().Interface("design", d).Msg("design loaded")

		cmd.design = d

		return nil
	}
}
