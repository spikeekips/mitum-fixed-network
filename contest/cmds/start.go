package cmds

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	dockerClient "github.com/docker/docker/client"
	"golang.org/x/xerrors"

	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/storage"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	networkName  = "contest-network"
	DefaultAlias = "test"
)

var conditionActions = map[string]contestlib.ConditionActionLoader{}

type StartCommand struct {
	Image      string        `help:"docker image for node runner (default: ${start_image})" default:"${start_image}"`
	RunnerPath string        `arg:"" name:"runner-path" help:"mitum node runner, 'mitum-runner' path" type:"existingfile"`
	Design     string        `arg:"" name:"node design file" help:"contest design file" type:"existingfile"`
	Alias      string        `arg:"" name:"alias of this test" help:"name of test" default:"${alias}" optional:""`
	Output     string        `help:"output directory" type:"existingdir"`
	ExitAfter  time.Duration `help:"exit after the given duration (default: ${exit_after})" default:"${exit_after}"`
	log        logging.Logger
	design     *contestlib.ContestDesign
	encs       *encoder.Encoders
	containers *contestlib.Containers
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
	if err := cmd.createContainers(dc); err != nil {
		return err
	} else {
		cmd.log.Debug().Msg("containers created")

		contestlib.ExitHooks.Add(func() {
			if err := cmd.containers.Kill("HUP"); err != nil {
				cmd.log.Error().Err(err).Msg("failed to kill containers") // NOTE ignore error
			}
		})
	}

	return cmd.run()
}

func (cmd *StartCommand) run() error {
	eventChan := make(chan *contestlib.Event, 10000)
	cmd.containers.SetEventChan(eventChan)

	if err := cmd.containers.Ready(); err != nil {
		return xerrors.Errorf("failed to ready containers: %w", err)
	}

	var mongodbClient *mongodbstorage.Client
	if client, err := mongodbstorage.NewClient(
		cmd.containers.StorageURI("events"), time.Second*2, time.Second*2,
	); err != nil {
		return xerrors.Errorf("failed to connect mongodb storage: %w", err)
	} else {
		mongodbClient = client
	}

	go cmd.handleEventChan(mongodbClient, eventChan)

	cmd.log.Debug().Msg("trying to run containers")

	if err := cmd.containers.Run(); err != nil {
		return err
	}

	if len(cmd.design.Conditions) < 1 {
		cmd.log.Debug().Msg("no conditions found")
	}

	return cmd.checkConditions(mongodbClient)
}

func (cmd *StartCommand) createContainers(dc *dockerClient.Client) error {
	var dockerNetworkID string
	if i, err := contestlib.CreateDockerNetwork(dc, networkName, false); err != nil {
		return xerrors.Errorf("failed to create new docker network: %w", err)
	} else {
		dockerNetworkID = i
	}

	cmd.Alias = strings.TrimSpace(cmd.Alias)
	if len(cmd.Alias) < 1 {
		cmd.Alias = DefaultAlias
	}

	output := filepath.Join(
		cmd.Output,
		cmd.Alias,
		fmt.Sprintf("%s-%s", time.Now().Format("2006-01-02T15-04-05"), Version),
	)
	if err := os.MkdirAll(output, 0o700); err != nil {
		return err
	}

	if err := cmd.copyFiles(output); err != nil {
		return xerrors.Errorf("failed to copy files: %w", err)
	}

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
		return err
	} else {
		cmd.containers = c
		_ = cmd.containers.SetLogger(cmd.log)
	}

	return nil
}

func (cmd *StartCommand) copyFiles(output string) error {
	files := [][]string{
		{cmd.RunnerPath, "runner"},
		{os.Args[0], "contest"},
	}

	for _, f := range files {
		if err := contestlib.CopyFile(f[0], filepath.Join(output, f[1]), 10000); err != nil {
			return err
		}
	}

	return nil
}

func (cmd *StartCommand) loadDesign(f string) error {
	conditionActions["start-node"] = cmd.defaultActionStartNode
	conditionActions["clean-storage"] = cmd.defaultActionCleanStorage

	if d, err := contestlib.LoadContestDesignFromFile(f, cmd.encs, conditionActions); err != nil {
		return xerrors.Errorf("failed to load design file: %w", err)
	} else if err := d.IsValid(nil); err != nil {
		return xerrors.Errorf("invalid design file: %w", err)
	} else {
		cmd.log.Debug().Interface("design", d).Msg("design loaded")

		cmd.design = d
	}

	if cmd.design.Config.Threshold < 67.0 {
		cmd.log.Warn().
			Float64("threshold", cmd.design.Config.Threshold).
			Msg("threshold is too low, recommend over 67.0")
	}

	return nil
}

func (cmd *StartCommand) handleEventChan(
	client *mongodbstorage.Client,
	eventChan chan *contestlib.Event,
) {
	for e := range eventChan {
		if r, err := e.Raw(); err != nil {
			cmd.log.Error().Err(err).Str("event", e.String()).Msg("malformed event found")

			continue
		} else if _, err := client.SetRaw("event", r); err != nil {
			cmd.log.Error().Err(err).Msg("failed to store event")

			continue
		}
	}
}

func (cmd *StartCommand) checkContainersIsRunning() chan struct{} {
	allEndedChan := make(chan struct{})
	go func() {
		var runnings bool

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for range ticker.C {
			rs, err := cmd.containers.Runnings()
			if err != nil {
				cmd.log.Error().Err(err).Msg("failed to check running containers")
			} else if len(rs) < 1 && runnings {
				allEndedChan <- struct{}{}
			}

			runnings = len(rs) > 0
		}
	}()

	return allEndedChan
}

func (cmd *StartCommand) checkConditions(client *mongodbstorage.Client) error {
	if cmd.log.IsVerbose() {
		for _, c := range cmd.design.Conditions {
			if c.Action() == nil {
				continue
			}

			if l, ok := c.Action().(logging.SetLogger); ok {
				_ = l.SetLogger(cmd.log)
			}
		}
	}

	exitChan := make(chan error, 10)
	allEndedChan := cmd.checkContainersIsRunning()

	cs := make([]*contestlib.Condition, len(cmd.design.Conditions))
	for i := range cmd.design.Conditions {
		cs[i] = cmd.design.Conditions[i]
	}

	cc := contestlib.NewConditionsChecker(client, "event", cs)
	_ = cc.SetLogger(cmd.log)

	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()

end:
	for {
		select {
		case err := <-exitChan:
			return err
		case <-allEndedChan:
			cmd.log.Debug().Msg("all containers ended")

			return nil
		case <-ticker.C:
			if passed, err := cc.Check(exitChan); err != nil {
				cmd.log.Error().Err(err).Msg("something wrong to check")
			} else if passed {
				cmd.log.Info().Msg("all conditions satisfied")

				break end
			}
		}
	}

	return nil
}

func (cmd *StartCommand) defaultActionStartNode(nodes []string) (func() error, error) {
	if len(nodes) < 1 {
		return nil, xerrors.Errorf("empty nodes to start")
	}

	return func() error {
		for _, n := range nodes {
			var found bool
			for _, d := range cmd.design.Nodes {
				if n == d.Address() {
					found = true
				}
			}

			if !found {
				return xerrors.Errorf("container name, %q not found", n)
			}
		}

		return cmd.containers.RunNodes(nodes)
	}, nil
}

func (cmd *StartCommand) defaultActionCleanStorage(nodes []string) (func() error, error) {
	if len(nodes) < 1 {
		return nil, xerrors.Errorf("empty nodes to clean storage")
	}

	return func() error {
		for _, n := range nodes {
			var found bool
			for _, d := range cmd.design.Nodes {
				if n == d.Address() {
					found = true
				}
			}

			if !found {
				return xerrors.Errorf("container name, %q not found", n)
			}
		}

		for _, node := range nodes {
			var st storage.Storage
			if s, err := cmd.containers.ContainerStorage(node); err != nil {
				return err
			} else {
				st = s
			}

			cmd.log.Debug().Str("node", node).Msg("trying to clean storage")
			if err := st.Clean(); err != nil {
				return err
			}

			cmd.log.Debug().Str("node", node).Msg("cleaned storage")
		}

		return nil
	}, nil
}
