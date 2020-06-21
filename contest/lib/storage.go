package contestlib

import (
	"net/url"
	"strings"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/storage"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util/encoder"
)

func LoadStorage(uri string, encs *encoder.Encoders) (storage.Storage, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, xerrors.Errorf("invalid storge uri, %q: %w", uri, err)
	}

	var st storage.Storage
	switch strings.ToLower(parsed.Scheme) {
	case "mongodb":
		if s, err := mongodbstorage.NewStorageFromURI(uri, encs); err != nil {
			return nil, err
		} else {
			st = s
		}
	default:
		return nil, xerrors.Errorf("failed to find storage by uri")
	}

	return st, nil
}
