package deploy

import (
	"context"

	"github.com/spikeekips/mitum/util"
)

var (
	ContextValueDeployKeyStorage util.ContextKey = "deploy_key_storage"
	ContextValueBlockDataCleaner util.ContextKey = "blockdata_cleaner"
	ContextValueDeployHandler    util.ContextKey = "deploy_handler"
)

func LoadDeployKeyStorageContextValue(ctx context.Context, l **DeployKeyStorage) error {
	return util.LoadFromContextValue(ctx, ContextValueDeployKeyStorage, l)
}

func LoadBlockDataCleanerContextValue(ctx context.Context, l **BlockDataCleaner) error {
	return util.LoadFromContextValue(ctx, ContextValueBlockDataCleaner, l)
}

func LoadDeployHandler(ctx context.Context, l **DeployHandlers) error {
	return util.LoadFromContextValue(ctx, ContextValueDeployHandler, l)
}
