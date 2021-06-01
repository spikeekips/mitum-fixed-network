package deploy

import (
	"context"

	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/storage"
)

var HookNameInitializeDeployKeyStorage = "initialize_deploy_key_storage"

func HookInitializeDeployKeyStorage(ctx context.Context) (context.Context, error) {
	var db storage.Database
	if err := process.LoadDatabaseContextValue(ctx, &db); err != nil {
		return ctx, err
	} else if i, err := NewDeployKeyStorage(db); err != nil {
		return ctx, err
	} else {
		return context.WithValue(ctx, ContextValueDeployKeyStorage, i), nil
	}
}
