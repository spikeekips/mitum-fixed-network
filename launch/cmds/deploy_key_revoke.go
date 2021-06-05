package cmds

import (
	"net/http"

	"github.com/spikeekips/mitum/launch/deploy"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
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
		return xerrors.Errorf("failed to initialize command: %w", err)
	}

	if err := cmd.requestToken(); err != nil {
		var pr network.Problem
		if xerrors.As(err, &pr) {
			cmd.Log().Error().Interface("problem", pr).Msg("failed")
		}

		return err
	}

	if err := cmd.requestRevoke(); err != nil {
		var pr network.Problem
		if xerrors.As(err, &pr) {
			cmd.Log().Error().Interface("problem", pr).Msg("failed")
		}

		return err
	}

	return nil
}

func (cmd *DeployKeyRevokeCommand) requestRevoke() error {
	path := deploy.QuicHandlerPathDeployKeyKeyPrefix + "/" + cmd.DeployKey

	var res *http.Response
	if i, c, err := cmd.requestWithToken(path, "DELETE"); err != nil {
		return xerrors.Errorf("failed to revoke deploy key: %w", err)
	} else {
		defer func() {
			_ = c()
			_ = i.Body.Close()
		}()

		res = i
	}

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
		return xerrors.Errorf("deploy key not found")
	default:
		cmd.Log().Debug().Interface("response", res).Msg("failed to request")

		return xerrors.Errorf("failed to revoke deploy key")
	}
}
