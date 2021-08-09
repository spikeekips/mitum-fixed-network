package cmds

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/launch/deploy"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
)

type DeployKeyKeysCommand struct {
	*baseDeployKeyCommand
}

func NewDeployKeyKeysCommand() DeployKeyKeysCommand {
	return DeployKeyKeysCommand{
		baseDeployKeyCommand: newBaseDeployKeyCommand("deploy-key-keys"),
	}
}

func (cmd *DeployKeyKeysCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
	}

	if err := cmd.requestToken(); err != nil {
		var pr network.Problem
		if errors.As(err, &pr) {
			cmd.Log().Error().Interface("problem", pr).Msg("failed")
		}

		return err
	}

	if err := cmd.requestKeys(); err != nil {
		var pr network.Problem
		if errors.As(err, &pr) {
			cmd.Log().Error().Interface("problem", pr).Msg("failed")
		}

		return err
	}

	return nil
}

func (cmd *DeployKeyKeysCommand) requestKeys() error { // nolint:dupl
	res, c, err := cmd.requestWithToken(deploy.QuicHandlerPathDeployKeyKeys, "GET")
	if err != nil {
		return errors.Wrap(err, "failed to request deploy keys")
	}
	defer func() {
		_ = c()
		_ = res.Body.Close()
	}()

	if res.StatusCode == http.StatusOK {
		i, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return errors.Wrap(err, "failed to read body")
		}
		_, _ = fmt.Fprintln(os.Stdout, string(i))

		return nil
	}

	if i, err := network.LoadProblemFromResponse(res); err == nil {
		cmd.Log().Debug().Interface("response", res).Interface("problem", i).Msg("failed to request")

		return i
	}

	cmd.Log().Debug().Interface("response", res).Msg("failed to request")

	return errors.Errorf("failed to request deploy keys")
}
