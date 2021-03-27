package mongodbstorage

import (
	"context"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

var defaultLimitWriteModels int = 200

func checkURI(uri string) (connstring.ConnString, error) {
	cs, err := connstring.Parse(uri)
	if err != nil {
		return connstring.ConnString{}, storage.WrapStorageError(err)
	}

	if len(cs.Database) < 1 {
		return connstring.ConnString{}, storage.WrapStorageError(xerrors.Errorf("empty database name in mongodb uri: '%v'", uri))
	}

	return cs, nil
}

func parseDurationFromQuery(query url.Values, key string, v time.Duration) (time.Duration, error) {
	if sl, found := query[key]; !found || len(sl) < 1 {
		return v, nil
	} else if s := sl[len(sl)-1]; len(strings.TrimSpace(s)) < 1 { // pop last one
		return v, nil
	} else if d, err := time.ParseDuration(s); err != nil {
		return 0, xerrors.Errorf("invalid %s value for mongodb: %w", key, err)
	} else {
		return d, nil
	}
}

func writeBulkModels(
	ctx context.Context,
	client *Client,
	col string,
	models []mongo.WriteModel,
	limit int,
	opts *options.BulkWriteOptions,
) (*mongo.BulkWriteResult, error) {
	if limit < 1 {
		limit = defaultLimitWriteModels
	}

	if len(models) < 1 {
		return nil, nil
	}

	result := new(mongo.BulkWriteResult)

	var ms []mongo.WriteModel
	var s, e int
	for {
		if e = s + limit; e > len(models) {
			e = len(models)
		}

		ms = models[s:e]
		if len(ms) < 1 {
			break
		}

		if res, err := client.Collection(col).BulkWrite(ctx, ms, opts); err != nil {
			if isDuplicatedError(err) {
				err = util.DuplicatedError.Wrap(err)
			}

			return nil, storage.WrapStorageError(err)
		} else {
			result.InsertedCount += res.InsertedCount
			result.MatchedCount += res.MatchedCount
			result.ModifiedCount += res.ModifiedCount
			result.DeletedCount += res.DeletedCount
			result.UpsertedCount += res.UpsertedCount

			for i := range res.UpsertedIDs {
				result.UpsertedIDs[i] = res.UpsertedIDs[i]
			}
		}

		if len(ms) < limit {
			break
		}

		s += limit
		if s >= len(models) {
			break
		}
	}

	return result, nil
}
