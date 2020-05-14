package cmds

import (
	"bytes"
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerNetwork "github.com/docker/docker/api/types/network"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"golang.org/x/xerrors"

	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	networkName  = "contest-network"
	label        = "contest"
	mongodbImgae = "mongo:latest"
)

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

	log.Debug().Strs("image", []string{cmd.Image, mongodbImgae}).Msg("trying to pull image")
	if err := contestlib.PullImages(dc, cmd.Image, mongodbImgae); err != nil {
		return err
	}

	log.Debug().Msg("trying to create docker network")

	if err := contestlib.CleanContainers(dc, label, log); err != nil {
		return err
	}

	return cmd.runContainer(design, dc, log, exitHooks)
}

func (cmd *StartCommand) runContainer(
	design *contestlib.ContestDesign,
	dc *dockerClient.Client,
	log logging.Logger,
	exitHooks *[]func(),
) error {
	var dockerNetworkID string
	if i, err := contestlib.CreateDockerNetwork(dc, networkName, false); err != nil {
		return xerrors.Errorf("failed to create new docker network: %w", err)
	} else {
		dockerNetworkID = i
	}

	if err := runMongodb(dc, dockerNetworkID, log); err != nil {
		return err
	}

	var cts *contestlib.Containers
	if c, err := contestlib.NewContainers(
		dc,
		cmd.Image,
		cmd.RunnerPath,
		networkName,
		dockerNetworkID,
		label,
		design,
	); err != nil {
		return err
	} else {
		cts = c
		_ = cts.SetLogger(log)

		if !cmd.NotClean {
			contestlib.AddExitHook(exitHooks, func() {
				_ = contestlib.CleanContainers(dc, label, log)
				_ = cts.Clean()
			})
		}
	}

	return cts.Run()
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

func runMongodb(dc *dockerClient.Client, dockerNetworkID string, log logging.Logger) error {
	log.Debug().Msg("trying to run mongodb")

	var id string
	if r, err := dc.ContainerCreate(context.Background(),
		&container.Config{
			Image:        mongodbImgae,
			Labels:       map[string]string{label: "mongodb"},
			ExposedPorts: nat.PortSet{"27017/tcp": struct{}{}},
		},
		&container.HostConfig{
			PortBindings: nat.PortMap{
				"27017/tcp": []nat.PortBinding{
					{HostIP: "", HostPort: "37017"},
				},
			},
		},
		&dockerNetwork.NetworkingConfig{
			EndpointsConfig: map[string]*dockerNetwork.EndpointSettings{
				networkName: {NetworkID: dockerNetworkID},
			},
		},
		"contest-mongodb",
	); err != nil {
		return xerrors.Errorf("failed to create mongodb container: %w", err)
	} else {
		id = r.ID
	}

	if err := dc.ContainerStart(context.Background(), id, types.ContainerStartOptions{}); err != nil {
		return xerrors.Errorf("failed to run mongodb container: %w", err)
	}

	opt := types.ContainerLogsOptions{ShowStdout: true, Follow: true}
	if out, err := dc.ContainerLogs(context.Background(), id, opt); err != nil {
		return xerrors.Errorf("failed to run mongodb container: %w", err)
	} else {
		buf := make([]byte, 1024)
		for {
			n, err := out.Read(buf)
			if bytes.Contains(buf[:n], []byte("waiting for connections on port ")) {
				break
			}

			if err == io.EOF {
				break
			}
		}
	}

	log.Debug().Msg("mongodb launched")

	return nil
}
