package mongodbstorage

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

func checkURI(uri string) (connstring.ConnString, error) {
	cs, err := connstring.Parse(uri)
	if err != nil {
		return connstring.ConnString{}, storage.WrapError(err)
	}

	if len(cs.Database) < 1 {
		return connstring.ConnString{}, storage.WrapError(xerrors.Errorf("empty database name in mongodb uri: '%v'", uri))
	}

	return cs, nil
}

func updateDBInURI(uri, db string) (string, error) {
	if _, err := checkURI(uri); err != nil {
		return "", err
	}

	var n string
	if parsed, err := url.Parse(uri); err != nil {
		return "", err
	} else {
		parsed.Path = db
		n = parsed.String()
	}

	return n, nil
}

func NewTempURI(uri, prefix string) (string, error) {
	tmp := fmt.Sprintf("%s_%s", prefix, util.UUID().String())

	cs, err := updateDBInURI(uri, tmp)
	if err != nil {
		return "", err
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
