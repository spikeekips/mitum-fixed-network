package contestlib

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	dockerNetwork "github.com/docker/docker/api/types/network"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-connections/nat"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v2"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
)

type Containers struct {
	*logging.Logging
	client          *dockerClient.Client
	image           string
	runner          string
	networkName     string
	dockerNetworkID string
	label           string
	design          *ContestDesign
	tmp             string
	containers      []*Container
}

func NewContainers(
	dc *dockerClient.Client,
	image,
	runner,
	networkName,
	dockerNetworkID,
	label string,
	design *ContestDesign,
) (*Containers, error) {
	tmp, err := ioutil.TempDir("/tmp", "prefix")
	if err != nil {
		return nil, xerrors.Errorf("failed to create temp directory", err)
	}

	return &Containers{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "contest-containers")
		}),
		client:          dc,
		image:           image,
		runner:          runner,
		networkName:     networkName,
		dockerNetworkID: dockerNetworkID,
		label:           label,
		design:          design,
		tmp:             tmp,
	}, nil
}

func (cts *Containers) Clean() error {
	cts.Log().Debug().Msg("trying to clean")

	_ = os.RemoveAll(cts.tmp)

	return nil
}

func (cts *Containers) ports() ([]int, error) {
	var ports []int
	for range cts.design.Nodes {
		for {
			p, err := util.FreePort("udp")
			if err != nil {
				return nil, xerrors.Errorf("failed to find free port: %w", err)
			}

			var found bool
			for _, e := range ports {
				if p == e {
					found = true
					break
				}
			}

			if !found {
				ports = append(ports, p)
				break
			}
		}
	}

	return ports, nil
}

func (cts *Containers) createContainers() ([]*Container, error) {
	cts.Log().Debug().Str("tmp", cts.tmp).Msg("trying to create containers")

	var ports []int
	if p, err := cts.ports(); err != nil {
		return nil, err
	} else {
		ports = p
	}

	cs := make([]*Container, len(cts.design.Nodes))

	for i, d := range cts.design.Nodes {
		pk, _ := key.NewBTCPrivatekey()

		container := &Container{
			name:            d.Address,
			image:           cts.image,
			client:          cts.client,
			port:            ports[i],
			runner:          cts.runner,
			privatekey:      pk,
			networkName:     cts.networkName,
			dockerNetworkID: cts.dockerNetworkID,
			label:           cts.label,
		}

		address := d.Address
		container.Logging = logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "contest-container").Str("address", address)
		})
		_ = container.SetLogger(cts.Log())

		cts.Log().Debug().Str("address", d.Address).Msg("container created")

		cs[i] = container
	}

	return cs, nil
}

func (cts *Containers) create() error {
	var cs []*Container
	if c, err := cts.createContainers(); err != nil {
		return err
	} else {
		cs = c
	}

	rds := make([]*RemoteDesign, len(cs))
	for i, c := range cs {
		rds[i] = &RemoteDesign{
			Address:         c.name,
			PublickeyString: jsonencoder.ToString(c.privatekey.Publickey()),
			Network:         c.networkDesign().Publish,
		}
	}

	for _, c := range cs {
		c.rds = rds

		if b, err := yaml.Marshal(c.nodeDesign()); err != nil {
			return xerrors.Errorf("failed to create node design: %w", err)
		} else {
			designFile := filepath.Join(cts.tmp, fmt.Sprintf("design-%s.yml", c.name))
			if err := ioutil.WriteFile(designFile, b, 0600); err != nil {
				return err
			}
			c.designFile = designFile
		}
	}

	cts.containers = cs

	return nil
}

func (cts *Containers) Run() error {
	if err := cts.create(); err != nil {
		return xerrors.Errorf("failed to create containers: %w", err)
	}

	errChan := make(chan error)
	doneChan := make(chan []interface{})

	go func() {
		var count int
		var occurred []interface{}
		for err := range errChan {
			count++
			if err != nil {
				occurred = append(occurred, err)
			}

			if len(cts.containers) == count {
				break
			}
		}

		doneChan <- occurred
	}()

	for _, c := range cts.containers {
		go func(c *Container) {
			err := c.run()
			if err != nil {
				cts.Log().Error().Err(err).Str("address", c.name).Msg("ended")
			} else {
				cts.Log().Debug().Str("address", c.name).Msg("ended")
			}

			errChan <- err
		}(c)
	}

	occurred := <-doneChan
	if len(occurred) < 1 {
		return nil
	}

	return xerrors.Errorf("failed to run containers")
}

type Container struct {
	*logging.Logging
	name            string
	image           string
	client          *dockerClient.Client
	port            int
	runner          string
	privatekey      key.Privatekey
	id              string
	networkName     string
	dockerNetworkID string
	label           string
	rds             []*RemoteDesign
	designFile      string
}

func (ct *Container) Name() string {
	return fmt.Sprintf("contest-node-%s", ct.name)
}

func (ct *Container) create() error {
	r, err := ct.client.ContainerCreate(
		context.Background(),
		ct.configCreate(),
		ct.configHost(),
		ct.configNetworking(),
		ct.Name(),
	)
	if err != nil {
		return xerrors.Errorf("failed to create container: %w", err)
	}

	ct.id = r.ID

	return nil
}

func (ct *Container) run() error {
	ct.Log().Debug().Msg("trying to run")

	if err := ct.create(); err != nil {
		return err
	}
	ct.Log().Debug().Msg("container created")

	if err := ct.client.ContainerStart(context.Background(), ct.id, types.ContainerStartOptions{}); err != nil {
		return xerrors.Errorf("failed to run container: %w", err)
	}
	ct.Log().Debug().Msg("container started")

	logChan := make(chan []byte)
	defer close(logChan)

	go func() {
		for b := range logChan {
			ct.Log().Debug().Str("msg", string(b)).Msg("container log")
		}
	}()

	if status, err := ContainerWait(ct.client, ct.id, logChan); err != nil {
		return err
	} else if status != 0 {
		return xerrors.Errorf("container exited abnormaly: statuscode=%d", status)
	}

	return nil
}

func (ct *Container) networkDesign() *NetworkDesign {
	return &NetworkDesign{
		Bind:    "0.0.0.0:54321",
		Publish: fmt.Sprintf("quic://%s:54321", ct.Name()),
	}
}

func (ct *Container) nodeDesign() NodeDesign {
	var nodes []*RemoteDesign // nolint
	for _, d := range ct.rds {
		if d.Address == ct.name {
			continue
		}

		nodes = append(nodes, d)
	}

	return NodeDesign{
		Address:          ct.name,
		PrivatekeyString: jsonencoder.ToString(ct.privatekey),
		Storage:          fmt.Sprintf("mongodb://contest-mongodb:27017/contest_%s", ct.name),
		Network:          ct.networkDesign(),
		Nodes:            nodes,
	}
}

func (ct *Container) configCreate() *container.Config {
	return &container.Config{
		Cmd: []string{
			"/contest-runner",
			"--log-level", "info",
			"run",
			"--event-log", "/tmp/e.log",
			"--verbose",
			"/design.yml",
		},
		WorkingDir: "/",
		Tty:        false,
		Image:      ct.image,
		Labels: map[string]string{
			ct.label: ct.name,
		},
		ExposedPorts: nat.PortSet{
			"54321/udp": struct{}{},
		},
	}
}

func (ct *Container) configHost() *container.HostConfig {
	return &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: ct.runner,
				Target: "/contest-runner",
			},
			{
				Type:   mount.TypeBind,
				Source: ct.designFile,
				Target: "/design.yml",
			},
			{ // BLOCK remove
				Type:   mount.TypeBind,
				Source: "/usr/bin/nc",
				Target: "/nc",
			},
		},
		PortBindings: nat.PortMap{
			"54321/udp": []nat.PortBinding{
				{
					HostIP:   "",
					HostPort: fmt.Sprintf("%d", ct.port),
				},
			},
		},
	}
}

func (ct *Container) configNetworking() *dockerNetwork.NetworkingConfig {
	return &dockerNetwork.NetworkingConfig{
		EndpointsConfig: map[string]*dockerNetwork.EndpointSettings{
			ct.networkName: {
				NetworkID: ct.dockerNetworkID,
			},
		},
	}
}

func PullImages(dc *dockerClient.Client, images ...string) error {
	for _, i := range images {
		if err := PullImage(dc, i); err != nil {
			return err
		}
	}

	return nil
}

func PullImage(dc *dockerClient.Client, image string) error {
	opt := types.ImageListOptions{
		Filters: filters.NewArgs(
			filters.Arg("reference", image),
		),
	}

	if s, err := dc.ImageList(context.Background(), opt); err != nil {
		return err
	} else if len(s) > 0 {
		return nil
	}

	r, err := dc.ImagePull(context.Background(), image, types.ImagePullOptions{})
	if err != nil {
		return xerrors.Errorf("failed to pull image: %w", err)
	}

	_ = jsonmessage.DisplayJSONMessagesStream(r, os.Stderr, os.Stderr.Fd(), true, nil)

	return nil
}

func CleanContainers(dc *dockerClient.Client, label string, log logging.Logger) error {
	log.Debug().Msg("trying to clean")

	opt := types.ContainerListOptions{
		All: true,
	}

	containers, err := dc.ContainerList(context.Background(), opt)
	if err != nil {
		return err
	}

	var founds []string
	for i := range containers {
		c := containers[i]
		if _, found := c.Labels[label]; found {
			founds = append(founds, c.ID)
		}
	}

	log.Debug().Msgf("found %d containers for contest", len(founds))

	optRemove := types.ContainerRemoveOptions{
		Force: true,
	}
	for _, c := range founds {
		if err := dc.ContainerRemove(context.Background(), c, optRemove); err != nil {
			return err
		}
	}

	log.Debug().Msg("cleaned")

	return nil
}

func CreateDockerNetwork(dc *dockerClient.Client, networkName string, createNew bool) (string, error) {
	var found string
	if l, err := dc.NetworkList(context.Background(), types.NetworkListOptions{}); err != nil {
		return "", err
	} else {
		for i := range l {
			n := l[i]
			if n.Name == networkName {
				found = n.ID
				break
			}
		}
	}

	if len(found) > 0 {
		if !createNew {
			return found, nil
		}

		if err := dc.NetworkRemove(context.Background(), found); err != nil {
			return "", err
		}
	}

	if r, err := dc.NetworkCreate(context.Background(), networkName, types.NetworkCreate{}); err != nil {
		return "", err
	} else {
		return r.ID, nil
	}
}

func ContainerWait(dc *dockerClient.Client, id string, logChan chan []byte /* log output chan */) (
	int64 /* status code */, error,
) {
	var out io.ReadCloser
	if o, err := dc.ContainerLogs(
		context.Background(),
		id,
		types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Tail:       "all",
			Follow:     true,
		},
	); err != nil {
		return 1, err
	} else {
		out = o
	}

	endedChan := make(chan struct{}, 1)
	defer func() {
		endedChan <- struct{}{}
	}()

	go func() {
		buf := make([]byte, 1024)

	end:
		for {
			select {
			case <-endedChan:
				break end
			default:
				n, err := out.Read(buf)
				if n > 7 {
					logChan <- bytes.TrimSpace(buf[8:n])
				}

				if err == io.EOF {
					break end
				}
			}
		}
	}()

	statusCh, errCh := dc.ContainerWait(context.Background(), id, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return 1, err
		}
	case sb := <-statusCh:
		if sb.Error != nil {
			return 1, xerrors.Errorf(sb.Error.Message)
		} else {
			return sb.StatusCode, nil
		}
	}

	return 0, nil
}

// BLOCK generate genesis block and others
// BLOCK set basic policy
/*{
	log.Debug().Msg("NodeRunner generated")

	if gg, err := isaac.NewGenesisBlockV0Generator(nr.Localstate(), nil); err != nil {
		log.Error().Err(err).Msg("failed to create genesis block generator")

		os.Exit(1)
	} else if blk, err := gg.Generate(); err != nil {
		log.Error().Err(err).Msg("failed to generate genesis block")

		os.Exit(1)
	} else {
		log.Info().Interface("block", blk).Msg("genesis block created")
	}
}*/

// distribute blocks to other nodes

// start nodes
