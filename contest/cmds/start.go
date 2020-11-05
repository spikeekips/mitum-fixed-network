package cmds

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	dockerClient "github.com/docker/docker/client"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launcher"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
)

var DefaultAlias = "test"

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
	exitChan   chan error
	eventChan  chan *contestlib.Event
}

func (cmd *StartCommand) Run(log logging.Logger) error {
	cmd.log = log

	if e, err := encoder.LoadEncoders(
		[]encoder.Encoder{jsonenc.NewEncoder(), bsonenc.NewEncoder()},
		contestlib.Hinters...,
	); err != nil {
		return xerrors.Errorf("failed to load encoders: %w", err)
	} else {
		cmd.encs = e
	}

	if err := cmd.loadDesign(cmd.Design); err != nil {
		return err
	}

	// create docker env
	var dc *dockerClient.Client
	if c, err := dockerClient.NewEnvClient(); err != nil {
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
	cmd.eventChan = make(chan *contestlib.Event, 10000)
	cmd.containers.SetEventChan(cmd.eventChan)

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

	go cmd.handleEventChan(mongodbClient)

	cmd.log.Debug().Msg("trying to run containers")

	if err := cmd.containers.Run(); err != nil {
		return err
	}

	if len(cmd.design.Conditions) < 1 {
		cmd.log.Debug().Msg("no conditions found")
	}

	cmd.exitChan = make(chan error, 10)

	return cmd.checkConditions(mongodbClient)
}

func (cmd *StartCommand) createContainers(dc *dockerClient.Client) error {
	var dockerNetworkID string
	if i, err := contestlib.CreateDockerNetwork(dc, false); err != nil {
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
	conditionActions["stop-contest"] = cmd.defaultActionStopContest
	conditionActions["build-blocks"] = cmd.defaultActionBuildBlocks
	conditionActions["mangle-blocks"] = cmd.defaultActionMangleBlocks

	if d, err := contestlib.LoadContestDesignFromFile(f, cmd.encs, conditionActions); err != nil {
		return xerrors.Errorf("failed to load design file: %w", err)
	} else if err := d.IsValid(nil); err != nil {
		return xerrors.Errorf("invalid design file: %w", err)
	} else {
		cmd.log.Debug().Interface("design", d).Msg("design loaded")

		cmd.design = d
	}

	if r := cmd.design.Config.GenesisPolicy.Policy().ThresholdRatio(); r < 67.0 {
		cmd.log.Warn().
			Float64("threshold", r.Float64()).
			Msg("threshold is too low, recommend over 67.0")
	}

	for _, a := range cmd.design.Conditions {
		if l, ok := a.Action().(logging.SetLogger); ok {
			_ = l.SetLogger(cmd.log)
		}
	}

	_ = cmd.design.Vars.
		Set("runner", cmd.RunnerPath).
		Set("network_id", string(cmd.design.Config.NetworkID()))

	return nil
}

func (cmd *StartCommand) handleEventChan(client *mongodbstorage.Client) {
	for e := range cmd.eventChan {
		if r, err := e.Raw(); err != nil {
			cmd.log.Error().Err(err).Str(contestlib.EvenCollection, e.String()).Msg("malformed event found")

			continue
		} else if _, err := client.AddRaw(contestlib.EvenCollection, r); err != nil {
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

	allEndedChan := cmd.checkContainersIsRunning()

	cs := make([]*contestlib.Condition, len(cmd.design.Conditions))
	for i := range cmd.design.Conditions {
		cs[i] = cmd.design.Conditions[i]
	}

	cc := contestlib.NewConditionsChecker(client, cs, cmd.design.Vars)
	_ = cc.SetLogger(cmd.log)

	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()

end:
	for {
		select {
		case err := <-cmd.exitChan:
			cmd.log.Error().Err(err).Msg("exit by error")

			return err
		case <-allEndedChan:
			cmd.log.Debug().Msg("all containers ended")

			return xerrors.Errorf("all containers ended")
		case <-ticker.C:
			if passed, err := cc.Check(cmd.exitChan); err != nil {
				cmd.log.Error().Err(err).Msg("something wrong to check")
			} else if passed {
				cmd.log.Info().Msg("all conditions satisfied")

				break end
			}
		}
	}

	return nil
}

func (cmd *StartCommand) defaultActionStartNode(nodes []string) (func(logging.Logger) error, error) {
	if len(nodes) < 1 {
		return nil, xerrors.Errorf("empty nodes to start")
	}

	return func(logging.Logger) error {
		for _, n := range nodes {
			var found bool
			for _, d := range cmd.design.Nodes {
				if n == d.Name {
					found = true
					break
				}
			}

			if !found {
				return xerrors.Errorf("container name, %q not found", n)
			}
		}

		return cmd.containers.RunNodes(nodes)
	}, nil
}

func (cmd *StartCommand) defaultActionCleanStorage(nodes []string) (func(logging.Logger) error, error) {
	if len(nodes) < 1 {
		return nil, xerrors.Errorf("empty nodes to clean storage")
	}

	return func(logging.Logger) error {
		var cs []*contestlib.Container
		for _, n := range nodes {
			if ct, found := cmd.containers.Container(n); !found {
				return xerrors.Errorf("container name, %q not found", n)
			} else {
				cs = append(cs, ct)
			}
		}

		for _, ct := range cs {
			if st, err := ct.Storage(false); err != nil {
				return err
			} else if err := st.Clean(); err != nil {
				return err
			} else {
				cmd.log.Debug().Str("node", ct.Name()).Msg("cleaned storage")
			}
		}

		return nil
	}, nil
}

func (cmd *StartCommand) defaultActionStopContest([]string) (func(logging.Logger) error, error) {
	return func(logging.Logger) error {
		cmd.exitChan <- xerrors.Errorf("stopped by design")

		return nil
	}, nil
}

func (cmd *StartCommand) defaultActionBuildBlocks(args []string) (func(logging.Logger) error, error) {
	var height base.Height
	if len(args) != 1 {
		return nil, xerrors.Errorf("one height must be given")
	} else if h, err := parseHeightFromString(args[0]); err != nil {
		return nil, err
	} else {
		height = h
	}

	return func(logging.Logger) error {
		cmd.log.Debug().Hinted("target_height", height).Msg("trying to build blocks")

		var genesis *isaac.Local
		var all []*isaac.Local
		for _, d := range cmd.design.Nodes {
			ct, found := cmd.containers.Container(d.Name)
			if !found {
				return xerrors.Errorf("container name, %q not found", d.Name)
			}

			l := ct.Local()
			all = append(all, l)

			if ct.Name() == cmd.containers.GenesisNode().Name() {
				genesis = l
			}
		}

		for _, l := range all {
			for _, r := range all {
				if l.Node().Address() == r.Node().Address() {
					continue
				}

				if err := l.Nodes().Add(r.Node()); err != nil {
					panic(err)
				}
			}
		}

		suffrage := launcher.NewRoundrobinSuffrage(genesis, 100)
		if err := suffrage.Initialize(); err != nil {
			return err
		}

		if bg, err := isaac.NewDummyBlocksV0Generator(genesis, height, suffrage, all); err != nil {
			return err
		} else if err := bg.Generate(false); err != nil {
			return err
		}

		cmd.log.Debug().Hinted("target_height", height).Msg("blocks generated")

		cmd.eventChan <- contestlib.EmptyEvent().
			Add("module", "contest-build-blocks").
			Add("height", height.Int64()).
			Add("m", "built blocks")

		return nil
	}, nil
}

func (cmd *StartCommand) defaultActionMangleBlocks(args []string) (func(logging.Logger) error, error) {
	if len(args) < 3 {
		return nil, xerrors.Errorf("height and nodes must be given")
	}

	var fromHeight, toHeight base.Height
	if h, err := parseHeightFromString(args[0]); err != nil {
		return nil, err
	} else {
		fromHeight = h
	}
	if h, err := parseHeightFromString(args[1]); err != nil {
		return nil, err
	} else {
		toHeight = h
	}

	nodeNames := args[2:]

	return func(logging.Logger) error {
		l := cmd.log.WithLogger(func(ctx logging.Context) logging.Emitter {
			return ctx.Strs("nodes", nodeNames).
				Hinted("from_height", fromHeight).
				Hinted("to_height", toHeight)
		})

		l.Debug().Msg("trying to mangle blocks")

		var all []*isaac.Local
		if a, err := cmd.prepareContainersForMangle(nodeNames, fromHeight); err != nil {
			return err
		} else {
			all = a
		}

		suffrage := launcher.NewRoundrobinSuffrage(all[0], 100)
		if err := suffrage.Initialize(); err != nil {
			return err
		}

		if bg, err := isaac.NewDummyBlocksV0Generator(all[0], toHeight, suffrage, all); err != nil {
			return err
		} else if err := bg.Generate(false); err != nil {
			return err
		}

		l.Debug().Msg("blocks mangled")

		cmd.eventChan <- contestlib.EmptyEvent().
			Add("module", "contest-mangle-blocks").
			Add("from_height", fromHeight).
			Add("to_height", toHeight).
			Add("nodes", nodeNames).
			Add("m", "mangled blocks")

		return nil
	}, nil
}

func (cmd *StartCommand) prepareContainersForMangle(nodeNames []string, fromHeight base.Height) (
	[]*isaac.Local,
	error,
) {
	all := make([]*isaac.Local, len(nodeNames))
	for i, n := range nodeNames {
		var ct *contestlib.Container
		if c, found := cmd.containers.Container(n); !found {
			return nil, xerrors.Errorf("container name, %q not found", n)
		} else {
			ct = c
		}

		if st, err := ct.Storage(true); err != nil {
			return nil, err
		} else if err := st.CleanByHeight(fromHeight + 1); err != nil {
			return nil, err
		} else if err := ct.BlockFS().CleanByHeight(fromHeight + 1); err != nil {
			return nil, err
		}

		all[i] = ct.Local()
	}

	for _, l := range all {
		for _, r := range all {
			if l.Node().Address() == r.Node().Address() {
				continue
			}

			if err := l.Nodes().Add(r.Node()); err != nil {
				panic(err)
			}
		}
	}

	return all, nil
}

func parseHeightFromString(s string) (base.Height, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return base.NilHeight, xerrors.Errorf("invalid height string, %v found: %w", s, err)
	}

	h := base.Height(n)
	if err := h.IsValid(nil); err != nil {
		return base.NilHeight, xerrors.Errorf("invalid height string, %v found: %w", s, err)
	} else if h <= base.PreGenesisHeight+1 {
		return base.NilHeight, xerrors.Errorf("height, %v was already built", h)
	}

	return h, nil
}
