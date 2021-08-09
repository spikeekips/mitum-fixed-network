package cache

import (
	"net/url"
	"time"

	"github.com/pkg/errors"
)

var DefaultCacheExpire = time.Hour

type Cache interface {
	Get(interface{}) (interface{}, error)
	Has(interface{}) bool
	Set(interface{}, interface{}, time.Duration) error
	Purge() error
	Remove(interface{}) bool
	New() (Cache, error)
}

func NewCacheFromURI(uri string) (Cache, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid uri of cache, %q", uri)
	}
	switch {
	case u.Scheme == "gcache":
		return NewGCacheWithQuery(u.Query())
	case u.Scheme == "dummy":
		return Dummy{}, nil
	default:
		return nil, errors.Errorf("not supported uri of cache, %q", uri)
	}
}
