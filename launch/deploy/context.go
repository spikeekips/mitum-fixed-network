package deploy

import (
	"context"

	"github.com/spikeekips/mitum/util"
)

var ContextValueDeployKeyStorage util.ContextKey = "deploy_key_storage"

func LoadDeployKeyStorageContextValue(ctx context.Context, l **DeployKeyStorage) error {
	return util.LoadFromContextValue(ctx, ContextValueDeployKeyStorage, l)
}
