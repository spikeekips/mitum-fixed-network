package cmds

import (
	"os"
	"path/filepath"
	"time"

	dockerClient "github.com/docker/docker/client"
	"golang.org/x/xerrors"

	contestlib "github.com/spikeekips/mitum/contest/lib"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/logging"
)

var networkName = "contest-network"

type StartCommand struct {
	Image      string        `help:"docker image for node runner (default: ${start_image})" default:"${start_image}"`
	Design     string        `arg:"" name:"node design file" help:"contest design file" type:"existingfile"`
	RunnerPath string        `arg:"" name:"runner-path" help:"mitum node runner, 'mitum-runner' path" type:"existingfile"`
	Output     string        `help:"output directory" type:"existingdir"`
	ExitAfter  time.Duration `help:"exit after the given duration (default: ${exit_after})" default:"${exit_after}"`
	log        logging.Logger
	design     *contestlib.ContestDesign
	encs       *encoder.Encoders
}

func (cmd *StartCommand) Run(log logging.Logger) error {
	cmd.log = log

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

		contestlib.ExitHooks.Add(func() {
			if err := cts.Kill("HUP"); err != nil {
				// cmd.log.Error().Err(err).Msg("failed to kill containers") // NOTE ignore error
			}
		})
	}

	return cmd.run(cts)
}

func (cmd *StartCommand) run(cts *contestlib.Containers) error {
	eventChan := make(chan *contestlib.Event, 10000)
	cts.SetEventChan(eventChan)

	if err := cts.Create(); err != nil {
		return xerrors.Errorf("failed to create containers: %w", err)
	} else if err := cts.RunStorage(); err != nil {
		return xerrors.Errorf("failed to run storage: %w", err)
	}

	exitChan := make(chan error, 10)
	go func() {
		cmd.log.Debug().Msg("trying to run containers")

		exitChan <- cts.Run()
	}()

	if client, err := mongodbstorage.NewClient(cts.StorageURI("events"), time.Second*2, time.Second*2); err != nil {
		return xerrors.Errorf("failed to connect mongodb storage: %w", err)
	} else {
		if len(cmd.design.Conditions) < 1 {
			return <-exitChan
		}

		return cmd.checkConditions(client, eventChan, exitChan)
	}
}

func (cmd *StartCommand) createContainers(dc *dockerClient.Client) (*contestlib.Containers, error) {
	var dockerNetworkID string
	if i, err := contestlib.CreateDockerNetwork(dc, networkName, false); err != nil {
		return nil, xerrors.Errorf("failed to create new docker network: %w", err)
	} else {
		dockerNetworkID = i
	}

	output := filepath.Join(cmd.Output, Version, time.Now().Format("2006-01-02T15-04-05"))
	if err := os.MkdirAll(output, 0o700); err != nil {
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
		cmd.ExitAfter,
	); err != nil {
		return nil, err
	} else {
		cts = c
		_ = cts.SetLogger(cmd.log)
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

func (cmd *StartCommand) checkConditions(
	client *mongodbstorage.Client,
	eventChan chan *contestlib.Event,
	exitChan chan error,
) error {
	go func() {
		for e := range eventChan {
			if r, err := e.Raw(); err != nil {
				cmd.log.Error().Err(err).Str("event", e.String()).Msg("malformed event found")

				continue
			} else if _, err := client.SetRaw("event", r); err != nil {
				cmd.log.Error().Err(err).Msg("failed to store event")

				continue
			}
		}

		for range eventChan { // NOTE not blocking eventChan
		}
	}()

	cs := make([]contestlib.Condition, len(cmd.design.Conditions))
	for i := range cmd.design.Conditions {
		cs[i] = *(cmd.design.Conditions[i])
	}

	cc := contestlib.NewConditionsChecker(client, "event", cs)
	if cmd.log.IsVerbose() {
		_ = cc.SetLogger(cmd.log)
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

end:
	for {
		select {
		case err := <-exitChan:
			return err
		case <-ticker.C:
			if passed, err := cc.Check(); err != nil {
				cmd.log.Error().Err(err).Msg("something wrong to check")
			} else if passed {
				cmd.log.Info().Msg("all conditions satisfied")

				break end
			}
		}
	}

	return nil
}
