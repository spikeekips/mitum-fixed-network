package process

import (
	"context"
	"fmt"

	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/storage"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util/logging"
)

const HookNameCleanTempMongodbDatabase = "clean_temp_mongodb_database"

func HookCleanTempMongodbDatabase(ctx context.Context) (context.Context, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var db storage.Database
	if err := LoadDatabaseContextValue(ctx, &db); err != nil {
		return ctx, err
	}

	st, ok := db.(*mongodbstorage.Database)
	if !ok {
		log.Log().Debug().
			Str("database", fmt.Sprintf("%T", db)).
			Msg("not mongodb storage database; skip clean temporary database")

		return ctx, nil
	}

	if err := mongodbstorage.CleanTemporayDatabase(st); err != nil {
		return ctx, err
	}

	return ctx, nil
}
