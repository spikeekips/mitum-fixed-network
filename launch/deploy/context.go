package deploy

import (
	"context"

	"github.com/spikeekips/mitum/util"
)

var (
	ContextValueDeployKeyStorage util.ContextKey = "deploy_key_storage"
	ContextValueBlockdataCleaner util.ContextKey = "blockdata_cleaner"
	ContextValueDeployHandler    util.ContextKey = "deploy_handler"
)

func LoadDeployKeyStorageContextValue(ctx context.Context, l **DeployKeyStorage) error {
	return util.LoadFromContextValue(ctx, ContextValueDeployKeyStorage, l)
}

func LoadBlockdataCleanerContextValue(ctx context.Context, l **BlockdataCleaner) error {
	return util.LoadFromContextValue(ctx, ContextValueBlockdataCleaner, l)
}

func LoadDeployHandler(ctx context.Context, l **DeployHandlers) error {
	return util.LoadFromContextValue(ctx, ContextValueDeployHandler, l)
}
