package cmds

import (
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
}

func (cmd *StartCommand) Run(log logging.Logger, exitHooks *[]func()) error {
	var encs *encoder.Encoders
	if e, err := contestlib.LoadEncoder(); err != nil {
		return xerrors.Errorf("failed to load encoders: %w", err)
	} else {
		encs = e
	}

	// load design file
	var design *contestlib.ContestDesign
	if d, err := loadDesign(cmd.Design, encs); err != nil {
		return err
	} else {
		design = d

		log.Debug().Interface("design", design).Msg("design loaded")
	}

	// create docker env
	var dc *dockerClient.Client
	if c, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv); err != nil {
		return err
	} else {
		dc = c
	}

	log.Debug().Strs("image", []string{cmd.Image, contestlib.MongodbImage}).Msg("trying to pull image")
	if err := contestlib.PullImages(dc, cmd.Image, contestlib.MongodbImage); err != nil {
		return err
	}

	log.Debug().Msg("trying to create docker network")

	if err := contestlib.CleanContainers(dc, log); err != nil {
		return err
	}

	log.Debug().Msg("trying to create containers")
	var cts *contestlib.Containers
	if c, err := cmd.createContainers(design, encs, dc, log, exitHooks); err != nil {
		return err
	} else {
		cts = c
		log.Debug().Msg("containers created")
	}

	log.Debug().Msg("trying to run containers")

	return cts.Run()
}

func (cmd *StartCommand) createContainers(
	design *contestlib.ContestDesign,
	encs *encoder.Encoders,
	dc *dockerClient.Client,
	log logging.Logger,
	exitHooks *[]func(),
) (*contestlib.Containers, error) {
	var dockerNetworkID string
	if i, err := contestlib.CreateDockerNetwork(dc, networkName, false); err != nil {
		return nil, xerrors.Errorf("failed to create new docker network: %w", err)
	} else {
		dockerNetworkID = i
	}

	var cts *contestlib.Containers
	if c, err := contestlib.NewContainers(
		dc,
		encs,
		cmd.Image,
		cmd.RunnerPath,
		networkName,
		dockerNetworkID,
		design,
	); err != nil {
		return nil, err
	} else {
		cts = c
		_ = cts.SetLogger(log)

		if !cmd.NotClean {
			contestlib.AddExitHook(exitHooks, func() {
				_ = contestlib.CleanContainers(dc, log)
				_ = cts.Clean()
			})
		}
	}

	return cts, nil
}

func loadDesign(f string, encs *encoder.Encoders) (*contestlib.ContestDesign, error) {
	if d, err := contestlib.LoadContestDesignFromFile(f, encs); err != nil {
		return nil, xerrors.Errorf("failed to load design file: %w", err)
	} else if err := d.IsValid(nil); err != nil {
		return nil, xerrors.Errorf("invalid design file: %w", err)
	} else {
		return d, nil
	}
}
