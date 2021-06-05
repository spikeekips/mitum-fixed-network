package cmds

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/spikeekips/mitum/launch/deploy"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
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
		return xerrors.Errorf("failed to initialize command: %w", err)
	}

	if err := cmd.requestToken(); err != nil {
		var pr network.Problem
		if xerrors.As(err, &pr) {
			cmd.Log().Error().Interface("problem", pr).Msg("failed")
		}

		return err
	}

	if err := cmd.requestKeys(); err != nil {
		var pr network.Problem
		if xerrors.As(err, &pr) {
			cmd.Log().Error().Interface("problem", pr).Msg("failed")
		}

		return err
	}

	return nil
}

func (cmd *DeployKeyKeysCommand) requestKeys() error { // nolint:dupl
	var res *http.Response
	if i, c, err := cmd.requestWithToken(deploy.QuicHandlerPathDeployKeyKeys, "GET"); err != nil {
		return xerrors.Errorf("failed to request deploy keys: %w", err)
	} else {
		defer func() {
			_ = c()
			_ = i.Body.Close()
		}()

		res = i
	}

	if res.StatusCode == http.StatusOK {
		if i, err := ioutil.ReadAll(res.Body); err != nil {
			return xerrors.Errorf("failed to read body: %w", err)
		} else {
			_, _ = fmt.Fprintln(os.Stdout, string(i))

			return nil
		}
	}

	if i, err := network.LoadProblemFromResponse(res); err == nil {
		cmd.Log().Debug().Interface("response", res).Interface("problem", i).Msg("failed to request")

		return i
	}

	cmd.Log().Debug().Interface("response", res).Msg("failed to request")

	return xerrors.Errorf("failed to request deploy keys")
}
