package contestlib

import (
	"bufio"
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
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	MongodbImage         = "mongo:latest"
	MongodbContainerName = "contest-mongodb"
	MongodbExternalPort  = "37017"
	ContainerLabel       = "contest"
)

var policyBody = `{
  "_hint": {
    "type": { "name": "policy-body-v0", "code": "0801" },
    "version": "0.0.1"
  },
  "threshold": [ %d, %f ],
  "timeout_waiting_proposal": 5000000000,
  "interval_broadcasting_init_ballot": 1000000000,
  "interval_broadcasting_proposal": 1000000000,
  "wait_broadcasting_accept_ballot": 2000000000,
  "interval_broadcasting_accept_ballot": 1000000000,
  "number_of_acting_suffrage_nodes": 1,
  "timespan_valid_ballot": 60000000000
}`

type Containers struct {
	*logging.Logging
	encs            *encoder.Encoders
	client          *dockerClient.Client
	image           string
	runner          string
	networkName     string
	dockerNetworkID string
	design          *ContestDesign
	tmp             string
	containers      []*Container
	mongodbIP       string
}

func NewContainers(
	dc *dockerClient.Client,
	encs *encoder.Encoders,
	image,
	runner,
	networkName,
	dockerNetworkID string,
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
		encs:            encs,
		image:           image,
		runner:          runner,
		networkName:     networkName,
		dockerNetworkID: dockerNetworkID,
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

	for i, c := range cs {
		c.rds = rds

		if b, err := yaml.Marshal(c.nodeDesign(i == 0)); err != nil {
			return xerrors.Errorf("failed to create node design: %w", err)
		} else {
			designFile := filepath.Join(cts.tmp, fmt.Sprintf("design-%s.yml", c.name))
			if err := ioutil.WriteFile(designFile, b, 0600); err != nil {
				return err
			}
			c.designFile = designFile
		}

		baseLogFile := filepath.Join(cts.tmp, fmt.Sprintf("base-%s.yml", c.name))
		if err := ioutil.WriteFile(baseLogFile, nil, 0600); err != nil {
			return err
		}
		c.baseLogFile = baseLogFile

		eventLogFile := filepath.Join(cts.tmp, fmt.Sprintf("event-%s.yml", c.name))
		if err := ioutil.WriteFile(eventLogFile, nil, 0600); err != nil {
			return err
		}
		c.eventLogFile = eventLogFile
	}

	cts.containers = cs

	return nil
}

func (cts *Containers) Run() error {
	if err := cts.create(); err != nil {
		return xerrors.Errorf("failed to create containers: %w", err)
	}

	if err := cts.runStorage(); err != nil {
		return xerrors.Errorf("failed to run storage: %w", err)
	}

	if err := cts.initializeByGenesisNode(); err != nil {
		return xerrors.Errorf("failed to initilaize by genesis node: %w", err)
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
			err := c.run(false)
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

func (cts *Containers) initializeByGenesisNode() error {
	cts.Log().Debug().Msg("trying to initialize")

	gnode := cts.containers[0]

	if err := gnode.run(true); err != nil {
		return err
	} else if err := CleanContainer(cts.client, gnode.id, cts.Log()); err != nil {
		return err
	}

	var gstorage storage.Storage
	if st, err := gnode.Storage(cts.encs); err != nil {
		return err
	} else {
		gstorage = st
	}

	cts.Log().Debug().Msg("sync storage")
	for _, c := range cts.containers {
		if c.name == gnode.name {
			continue
		}

		cts.Log().Debug().Str("from", gnode.name).Str("to", c.name).Msg("trying to sync storage")
		if st, err := c.Storage(cts.encs); err != nil {
			return err
		} else if err := st.Copy(gstorage); err != nil {
			return err
		}

		cts.Log().Debug().Str("from", gnode.name).Str("to", c.name).Msg("storage synced")
	}

	cts.Log().Debug().Msg("initialized")

	return nil
}

func (cts *Containers) runStorage() error {
	cts.Log().Debug().Msg("trying to run mongodb")

	var id string
	if r, err := cts.client.ContainerCreate(context.Background(),
		&container.Config{
			Image:        MongodbImage,
			Labels:       map[string]string{ContainerLabel: "mongodb"},
			ExposedPorts: nat.PortSet{"27017/tcp": struct{}{}},
		},
		&container.HostConfig{
			PortBindings: nat.PortMap{
				"27017/tcp": []nat.PortBinding{
					{HostIP: "", HostPort: MongodbExternalPort},
				},
			},
		},
		&dockerNetwork.NetworkingConfig{
			EndpointsConfig: map[string]*dockerNetwork.EndpointSettings{
				cts.networkName: {NetworkID: cts.dockerNetworkID},
			},
		},
		MongodbContainerName,
	); err != nil {
		return xerrors.Errorf("failed to create mongodb container: %w", err)
	} else {
		id = r.ID
	}

	if err := cts.client.ContainerStart(context.Background(), id, types.ContainerStartOptions{}); err != nil {
		return xerrors.Errorf("failed to run mongodb container: %w", err)
	}

	if r, err := ContainerInspect(cts.client, id); err != nil {
		return err
	} else {
		cts.mongodbIP = r.NetworkSettings.IPAddress

		for _, c := range cts.containers {
			c.mongodbIP = cts.mongodbIP
		}
	}

	if err := ContainerWaitCheck(cts.client, id, "waiting for connections on port", 100); err != nil {
		return xerrors.Errorf("failed to run mongodb container: %w", err)
	}

	cts.Log().Debug().Msg("mongodb launched")

	return nil
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
	rds             []*RemoteDesign
	designFile      string
	eventLogFile    string
	baseLogFile     string
	mongodbIP       string
}

func (ct *Container) Name() string {
	return fmt.Sprintf("contest-node-%s", ct.name)
}

func (ct *Container) create() error {
	r, err := ct.client.ContainerCreate(
		context.Background(),
		ct.configCreate([]string{
			"/contest-runner",
			"--log", "/base.log",
			"--log-level", "info",
			"run",
			"--event-log", "/event.log",
			"--verbose",
			"/design.yml",
		}),
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

func (ct *Container) createInitialize() error {
	r, err := ct.client.ContainerCreate(
		context.Background(),
		ct.configCreate([]string{
			"/contest-runner",
			"--log-level", "info",
			"init",
			"--verbose",
			"/design.yml",
		}),
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

func (ct *Container) run(initialize bool) error {
	l := ct.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Bool("initialize", initialize)
	})
	l.Debug().Msg("trying to run")

	var createFunc func() error
	if initialize {
		createFunc = ct.createInitialize
	} else {
		createFunc = ct.create
	}

	if err := createFunc(); err != nil {
		return err
	}
	l.Debug().Msg("container created")

	if err := ct.client.ContainerStart(context.Background(), ct.id, types.ContainerStartOptions{}); err != nil {
		return xerrors.Errorf("failed to run container: %w", err)
	}
	l.Debug().Msg("container started")

	logChan := make(chan []byte)
	defer close(logChan)

	go func() {
		for b := range logChan {
			l.Debug().Str("msg", string(b)).Msg("container log")
		}
	}()

	if status, err := ContainerWait(ct.client, ct.id, logChan); err != nil {
		return err
	} else if status != 0 {
		return xerrors.Errorf("container exited abnormally: statuscode=%d", status)
	}

	l.Debug().Msg("ended")

	return nil
}

func (ct *Container) networkDesign() *NetworkDesign {
	return &NetworkDesign{
		Bind:    "0.0.0.0:54321",
		Publish: fmt.Sprintf("quic://%s:54321", ct.Name()),
	}
}

func (ct *Container) nodeDesign(isGenesisNode bool) NodeDesign {
	var nodes []*RemoteDesign // nolint
	for _, d := range ct.rds {
		if d.Address == ct.name {
			continue
		}

		nodes = append(nodes, d)
	}

	nd := NodeDesign{
		Address:          ct.name,
		PrivatekeyString: jsonencoder.ToString(ct.privatekey),
		Storage:          ct.storageURIInternal(),
		Network:          ct.networkDesign(),
		Nodes:            nodes,
	}

	nd.GenesisOperations = ct.genesisOperationDesign(isGenesisNode)

	return nd
}

func (ct *Container) genesisOperationDesign(isGenesisNode bool) []*OperationDesign {
	if !isGenesisNode {
		return nil
	}

	var ops []*OperationDesign
	if isGenesisNode {
		ops = append(ops, &OperationDesign{
			BodyString: fmt.Sprintf(
				policyBody,
				len(ct.rds),
				67.0,
			),
		})
	}

	return ops
}

func (ct *Container) configCreate(cmd []string) *container.Config {
	return &container.Config{
		Cmd:        cmd,
		WorkingDir: "/",
		Tty:        false,
		Image:      ct.image,
		Labels: map[string]string{
			ContainerLabel: ct.name,
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
			{
				Type:   mount.TypeBind,
				Source: ct.baseLogFile,
				Target: "/base.log",
			},
			{
				Type:   mount.TypeBind,
				Source: ct.eventLogFile,
				Target: "/event.log",
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

func (ct *Container) storageURIInternal() string {
	return fmt.Sprintf("mongodb://%s:27017/contest_%s", MongodbContainerName, ct.name)
}

func (ct *Container) storageURIExternal() string {
	return fmt.Sprintf("mongodb://%s:%s/contest_%s", ct.mongodbIP, MongodbExternalPort, ct.name)
}

func (ct *Container) Storage(encs *encoder.Encoders) (storage.Storage, error) {
	return LoadStorage(ct.storageURIExternal(), encs)
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

func CleanContainers(dc *dockerClient.Client, log logging.Logger) error {
	log.Debug().Msg("trying to clean containers")

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
		if _, found := c.Labels[ContainerLabel]; found {
			founds = append(founds, c.ID)
		}
	}

	log.Debug().Msgf("found %d containers for contest", len(founds))

	if len(founds) < 1 {
		log.Debug().Msg("nothing to be cleaned")

		return nil
	}

	for _, id := range founds {
		if err := CleanContainer(dc, id, log); err != nil {
			return err
		}
	}

	log.Debug().Msg("containers cleaned")

	return nil
}

func CleanContainer(dc *dockerClient.Client, id string, log logging.Logger) error {
	l := log.WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("id", id)
	})

	l.Debug().Msg("trying to clean container")

	optRemove := types.ContainerRemoveOptions{
		Force: true,
	}
	if err := dc.ContainerRemove(context.Background(), id, optRemove); err != nil {
		return err
	}

	l.Debug().Msg("container cleaned")

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
	if o, err := dc.ContainerLogs(context.Background(), id, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "all",
		Follow:     true,
	}); err != nil {
		return 1, err
	} else {
		out = o
	}

	endedChan := make(chan struct{}, 1)
	defer func() {
		endedChan <- struct{}{}
	}()

	go func() {
		reader := bufio.NewReader(out)

	end:
		for {
			select {
			case <-endedChan:
				break end
			default:
				for {
					l, err := reader.ReadBytes('\n')
					if len(l) > 7 {
						logChan <- bytes.TrimSpace(l[8:])
					}

					if err != nil {
						break end
					}
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
		var err error
		if sb.Error != nil {
			err = xerrors.Errorf(sb.Error.Message)
		}

		return sb.StatusCode, err
	}

	return 0, nil
}

func ContainerInspect(dc *dockerClient.Client, id string) (types.ContainerJSON, error) {
	return dc.ContainerInspect(context.Background(), id)
}

func ContainerWaitCheck(dc *dockerClient.Client, id, s string, limit int) error {
	if limit < 1 {
		limit = 100
	}

	nd := []byte(s)

	var out io.ReadCloser

	opt := types.ContainerLogsOptions{ShowStdout: true, Follow: true}
	if o, err := dc.ContainerLogs(context.Background(), id, opt); err != nil {
		return xerrors.Errorf("failed to get container logs: %w", err)
	} else {
		out = o
	}

	var found bool
	var count int
	buf := make([]byte, 1024)
	for {
		if count > limit {
			break
		}

		n, err := out.Read(buf)
		count++
		if bytes.Contains(buf[:n], nd) {
			found = true
			break
		}

		if err == io.EOF {
			break
		}
	}

	if !found {
		return xerrors.Errorf("not found from logs")
	}

	return nil
}
