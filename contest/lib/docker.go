package contestlib

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/logging"
)

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

func FindLabeledContainers(dc *dockerClient.Client, label string) ([]string, error) {
	containers, err := dc.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	var founds []string
	for i := range containers {
		c := containers[i]
		if _, found := c.Labels[label]; found {
			founds = append(founds, c.ID)
		}
	}

	return founds, nil
}

func StopContainers(dc *dockerClient.Client) error {
	var founds []string
	if l, err := FindLabeledContainers(dc, ContainerLabel); err != nil {
		return err
	} else {
		founds = l
	}

	if len(founds) < 1 {
		return nil
	}

	for _, id := range founds {
		if err := StopContainer(dc, id); err != nil {
			return err
		}
	}

	return nil
}

func KillContainers(dc *dockerClient.Client, sig string) error {
	var founds []string
	if l, err := FindLabeledContainers(dc, ContainerLabel); err != nil {
		return err
	} else {
		founds = l
	}

	if len(founds) < 1 {
		return nil
	}

	var errs []error
	for _, id := range founds {
		if err := KillContainer(dc, id, sig); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return xerrors.Errorf("failed to kill containers: %v", errs)
	}

	return nil
}

func KillContainer(dc *dockerClient.Client, id, sig string) error {
	return dc.ContainerKill(context.Background(), id, sig)
}

func StopContainer(dc *dockerClient.Client, id string) error {
	if err := dc.ContainerStop(context.Background(), id, nil); err != nil {
		return err
	}

	return nil
}

func CleanContainers(dc *dockerClient.Client, log logging.Logger) error {
	log.Debug().Msg("trying to clean containers")

	var founds []string
	if l, err := FindLabeledContainers(dc, ContainerLabel); err != nil {
		return err
	} else {
		founds = l
	}

	log.Debug().Msgf("found %d containers for contest", len(founds))

	if len(founds) < 1 {
		log.Debug().Msg("nothing to be cleaned")

		return nil
	}

	for _, id := range founds {
		if err := StopContainer(dc, id); err != nil {
			return err
		}

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

func ContainerWait(dc *dockerClient.Client, id string) (
	int64 /* status code */, error,
) {
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

func ContainerIsRunning(dc *dockerClient.Client, id string) (bool, error) {
	if res, err := ContainerInspect(dc, id); err != nil {
		return false, err
	} else {
		return res.State.Running, nil
	}
}

func ContainerInspect(dc *dockerClient.Client, id string) (types.ContainerJSON, error) {
	return dc.ContainerInspect(context.Background(), id)
}

func ContainerWaitCheck(dc *dockerClient.Client, id, s string, limit int) error {
	if limit < 1 {
		limit = 100
	}

	nd := []byte(s)

	outChan := make(chan []byte)
	if cancel, err := ContainerLogs(dc, id, outChan, true, false); err != nil {
		return err
	} else {
		defer cancel()
	}

	var found bool
	var count int
	for o := range outChan {
		if count > limit {
			break
		}

		count++
		if bytes.Contains(o, nd) {
			found = true
			break
		}
	}

	if !found {
		return xerrors.Errorf("not found from logs")
	}

	return nil
}

func ContainerLogs(
	dc *dockerClient.Client,
	id string,
	outChan chan []byte,
	showStdout,
	showStderr bool,
) (func(), error) {
	var out io.ReadCloser
	if o, err := dc.ContainerLogs(context.Background(), id, types.ContainerLogsOptions{
		ShowStdout: showStdout,
		ShowStderr: showStderr,
		Tail:       "all",
		Follow:     true,
	}); err != nil {
		return nil, err
	} else {
		out = o
	}

	endedChan := make(chan struct{}, 1)

	go func() {
	end:
		for {
			select {
			case <-endedChan:
				break end
			default:
				h := make([]byte, 8)
				if _, err := out.Read(h); err != nil {
					break end
				}

				count := binary.BigEndian.Uint32(h[4:])
				l := make([]byte, count)
				if _, err := out.Read(l); err != nil {
					break end
				}

				outChan <- bytes.TrimSpace(l)
			}
		}
	}()

	return func() {
		endedChan <- struct{}{}
	}, nil
}

func ContainersPrune(dc *dockerClient.Client) error {
	if _, err := dc.ContainersPrune(context.Background(), filters.Args{}); err != nil {
		return err
	}

	return nil
}

func VolumesPrune(dc *dockerClient.Client) error {
	if _, err := dc.VolumesPrune(context.Background(), filters.Args{}); err != nil {
		return err
	}

	return nil
}
