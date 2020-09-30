package cache

import (
	"net/url"
	"time"

	"golang.org/x/xerrors"
)

var DefaultCacheExpire = time.Hour

type Cache interface {
	Get(interface{}) (interface{}, error)
	Has(interface{}) bool
	Set(interface{}, interface{}, time.Duration) error
	Purge() error
	New() (Cache, error)
}

func NewCacheFromURI(uri string) (Cache, error) {
	if u, err := url.Parse(uri); err != nil {
		return nil, xerrors.Errorf("invalid uri of cache, %q: %w", uri, err)
	} else {
		switch {
		case u.Scheme == "gcache":
			return NewGCacheWithQuery(u.Query())
		case u.Scheme == "dummy":
			return Dummy{}, nil
		default:
			return nil, xerrors.Errorf("unsupported uri of cache, %q", uri)
		}
	}
}
