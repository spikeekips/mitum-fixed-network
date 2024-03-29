package process

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/util"
)

const HookNameValidateConfig = "validate_config"

func HookValidateConfig(ctx context.Context) (context.Context, error) {
	if va, err := config.NewValidator(ctx); err != nil {
		return ctx, err
	} else if err := util.NewChecker("config-validator", []util.CheckerFunc{
		va.CheckNodeAddress,
		va.CheckNodePrivatekey,
		va.CheckNetworkID,
		va.CheckLocalNetwork,
		va.CheckStorage,
		va.CheckPolicy,
		va.CheckNodes,
		va.CheckSuffrage,
		va.CheckProposalProcessor,
		va.CheckGenesisOperations,
		va.CheckLocalConfig,
	}).Check(); err != nil {
		if errors.Is(err, util.IgnoreError) {
			return ctx, nil
		}

		return ctx, err
	} else {
		return va.Context(), nil
	}
}
