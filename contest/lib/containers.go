package contestlib

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	dockerNetwork "github.com/docker/docker/api/types/network"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

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
  "wait_broadcasting_accept_ballot": 5000000000,
  "interval_broadcasting_accept_ballot": 5000000000,
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
	outputDir       string
	containers      []*Container
	mongodbIP       string
	exitAfter       time.Duration
	eventChan       chan *Event
}

func NewContainers(
	dc *dockerClient.Client,
	encs *encoder.Encoders,
	image,
	runner,
	networkName,
	dockerNetworkID string,
	design *ContestDesign,
	outputDir string,
	exitAfter time.Duration,
) (*Containers, error) {
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
		outputDir:       outputDir,
		exitAfter:       exitAfter,
	}, nil
}

func (cts *Containers) Kill(sig string) error {
	cts.Log().Debug().Msg("trying to kill")

	if err := KillContainers(cts.client, sig); err != nil {
		return err
	}
	cts.Log().Debug().Msg("containers killed")

	return nil
}

func (cts *Containers) Stop() error {
	cts.Log().Debug().Msg("trying to stop")

	if err := StopContainers(cts.client); err != nil {
		return err
	}
	cts.Log().Debug().Msg("containers stoped")

	return nil
}

func (cts *Containers) Clean() error {
	cts.Log().Debug().Msg("trying to clean")

	if err := StopContainers(cts.client); err != nil {
		return err
	}
	cts.Log().Debug().Msg("containers stoped")

	if err := CleanContainers(cts.client, cts.Log()); err != nil {
		return err
	}
	cts.Log().Debug().Msg("containers cleaned")

	if err := ContainersPrune(cts.client); err != nil {
		return err
	}
	cts.Log().Debug().Msg("containers pruned")

	if err := VolumesPrune(cts.client); err != nil {
		return err
	}
	cts.Log().Debug().Msg("volumes pruned")

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
	cts.Log().Debug().Str("outputDir", cts.outputDir).Msg("trying to create containers")

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
			name:            d.Address(),
			image:           cts.image,
			client:          cts.client,
			port:            ports[i],
			runner:          cts.runner,
			privatekey:      pk,
			networkName:     cts.networkName,
			dockerNetworkID: cts.dockerNetworkID,
			exitAfter:       cts.exitAfter,
			eventChan:       cts.eventChan,
		}

		address := d.Address()
		container.Logging = logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "contest-container").Str("address", address)
		})
		_ = container.SetLogger(cts.Log())

		cs[i] = container
	}

	return cs, nil
}

func (cts *Containers) Create() error {
	if err := CopyFile(cts.runner, filepath.Join(cts.outputDir, "runner"), 10000); err != nil {
		return err
	}

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
			designFile := filepath.Join(cts.outputDir, fmt.Sprintf("design-%s.yml", c.name))
			if err := ioutil.WriteFile(designFile, b, 0o600); err != nil {
				return err
			}
			c.designFile = designFile
		}

		c.shareDir = cts.outputDir

		errWriter := filepath.Join(cts.outputDir, fmt.Sprintf("error-%s.log", c.name))
		if f, err := os.Create(errWriter); err != nil {
			return xerrors.Errorf("failed to create error log file: %w", err)
		} else {
			c.errWriter = f
		}
	}

	cts.containers = cs

	return nil
}

func (cts *Containers) Run() error {
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

func (cts *Containers) RunStorage() error {
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

func (cts *Containers) SetEventChan(ch chan *Event) {
	cts.eventChan = ch
}

func (cts *Containers) StorageURI(db string) string {
	return fmt.Sprintf("mongodb://%s:%s/%s", cts.mongodbIP, MongodbExternalPort, db)
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
	errWriter       io.Writer
	mongodbIP       string
	shareDir        string
	exitAfter       time.Duration
	eventChan       chan *Event
}

func (ct *Container) Name() string {
	return fmt.Sprintf("contest-node-%s", ct.name)
}

func (ct *Container) create() error {
	r, err := ct.client.ContainerCreate(
		context.Background(),
		ct.configCreate([]string{
			"/runner",
			"--log", "/dev/stdout",
			"--log", fmt.Sprintf("/share/base-%s.log", ct.name),
			"run",
			"--mem-prof", fmt.Sprintf("/share/%s-mem.prof", ct.name),
			"--cpu-prof", fmt.Sprintf("/share/%s-cpu.prof", ct.name),
			"--trace-prof", fmt.Sprintf("/share/%s-trace.prof", ct.name),
			"--exit-after", ct.exitAfter.String(),
			fmt.Sprintf("/share/design-%s.yml", ct.name),
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
			"/runner",
			"init",
			fmt.Sprintf("/share/design-%s.yml", ct.name),
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

	if cancel, err := ct.containerErr(); err != nil {
		return err
	} else {
		defer cancel()
	}

	if cancel, err := ct.containerLog(); err != nil {
		return err
	} else {
		defer cancel()
	}

	if status, err := ContainerWait(ct.client, ct.id); err != nil {
		return err
	} else if status != 0 {
		return xerrors.Errorf("container exited abnormally: statuscode=%d", status)
	}

	l.Debug().Msg("ended")

	return nil
}

func (ct *Container) containerLog() (func(), error) {
	if ct.eventChan == nil {
		return func() {}, nil
	}

	logChan := make(chan []byte)

	// stdout: event log
	var cancel func()
	if c, err := ContainerLogs(ct.client, ct.id, logChan, true, false); err != nil {
		return nil, err
	} else {
		cancel = func() {
			c()
			close(logChan)
		}
	}

	go func() {
		for b := range logChan {
			if e, err := NewEvent(b); err != nil {
				ct.Log().Error().Err(err).Msg("failed to parse event log")
			} else {
				ct.eventChan <- e.Add("_node", ct.name)
			}
		}
	}()

	return cancel, nil
}

func (ct *Container) containerErr() (func(), error) {
	errChan := make(chan []byte)
	var cancel func()
	if c, err := ContainerLogs(ct.client, ct.id, errChan, false, true); err != nil {
		return nil, err
	} else {
		cancel = func() {
			c()
			close(errChan)
		}
	}

	var errWriter io.Writer = ct.errWriter
	if ct.errWriter == nil {
		errWriter = ioutil.Discard
	}

	go func() {
		for b := range errChan {
			ct.Log().Error().Str("msg", string(b)).Msg("container error")
			if _, err := fmt.Fprintln(errWriter, string(b)); err != nil {
				ct.Log().Error().Err(err).Msg("failed to write error log")
			}
		}
	}()

	return cancel, nil
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
				Type:     mount.TypeBind,
				Source:   filepath.Join(ct.shareDir, "runner"),
				Target:   "/runner",
				ReadOnly: true,
			},
			{
				Type:   mount.TypeBind,
				Source: ct.shareDir,
				Target: "/share",
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
