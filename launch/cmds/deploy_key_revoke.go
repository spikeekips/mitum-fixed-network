package cmds

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/launch/deploy"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
)

type DeployKeyRevokeCommand struct {
	DeployKey string `arg:"" name:"deploy key"`
	*baseDeployKeyCommand
}

func NewDeployKeyRevokeCommand() DeployKeyRevokeCommand {
	return DeployKeyRevokeCommand{
		baseDeployKeyCommand: newBaseDeployKeyCommand("deploy-key-revoke"),
	}
}

func (cmd *DeployKeyRevokeCommand) Run(version util.Version) error {
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

	if err := cmd.requestRevoke(); err != nil {
		var pr network.Problem
		if errors.As(err, &pr) {
			cmd.Log().Error().Interface("problem", pr).Msg("failed")
		}

		return err
	}

	return nil
}

func (cmd *DeployKeyRevokeCommand) requestRevoke() error {
	path := deploy.QuicHandlerPathDeployKeyKeyPrefix + "/" + cmd.DeployKey

	res, c, err := cmd.requestWithToken(path, "DELETE")
	if err != nil {
		return errors.Wrap(err, "failed to revoke deploy key")
	}
	defer func() {
		_ = c()
		_ = res.Body.Close()
	}()

	if res.StatusCode == http.StatusOK {
		cmd.Log().Info().Str("deploy_key", cmd.DeployKey).Msg("deploy key revoked")

		return nil
	}

	if i, err := network.LoadProblemFromResponse(res); err == nil {
		cmd.Log().Debug().Interface("response", res).Interface("problem", i).Msg("failed to request")

		return i
	}

	cmd.Log().Debug().Interface("response", res).Msg("failed to request")

	switch res.StatusCode {
	case http.StatusNotFound:
		return errors.Errorf("deploy key not found")
	default:
		cmd.Log().Debug().Interface("response", res).Msg("failed to request")

		return errors.Errorf("failed to revoke deploy key")
	}
}
