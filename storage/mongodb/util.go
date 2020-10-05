package mongodbstorage

import (
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/storage"
)

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
