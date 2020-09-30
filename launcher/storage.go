package launcher

import (
	"net/url"
	"strings"

	"github.com/spikeekips/mitum/storage"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/encoder"
	"golang.org/x/xerrors"
)

func LoadStorage(uri string, encs *encoder.Encoders, ca cache.Cache) (storage.Storage, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, xerrors.Errorf("invalid storge uri, %q: %w", uri, err)
	}

	var st storage.Storage
	switch strings.ToLower(parsed.Scheme) {
	case "mongodb":
		if s, err := mongodbstorage.NewStorageFromURI(uri, encs, ca); err != nil {
			return nil, err
		} else {
			st = s
		}
	default:
		return nil, xerrors.Errorf("failed to find storage by uri")
	}

	return st, nil
}
