package cache

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type testGCache struct {
	suite.Suite
}

func (t *testGCache) TestNew() {
	ca, err := NewGCacheWithQuery(nil)
	t.NoError(err)

	_, ok := (interface{})(ca).(Cache)
	t.True(ok)

	t.Equal(DefaultGCacheSize, ca.size)
	t.Equal(DefaultCacheExpire, ca.expire)
}

func (t *testGCache) TestWithSize() {
	{
		query := url.Values{}
		query.Set("size", "a3333")
		_, err := NewGCacheWithQuery(query)
		t.Contains(err.Error(), "invalid size")
	}

	{
		query := url.Values{}
		query.Set("size", "3333")
		ca, err := NewGCacheWithQuery(query)
		t.NoError(err)

		t.Equal(3333, ca.size)
		t.Equal(DefaultCacheExpire, ca.expire)
	}
}

func (t *testGCache) TestWithExpire() {
	{
		query := url.Values{}
		query.Set("expire", "showme")
		_, err := NewGCacheWithQuery(query)
		t.Contains(err.Error(), "invalid expire")
	}

	{
		expire := time.Second * 3333

		query := url.Values{}
		query.Set("expire", expire.String())
		ca, err := NewGCacheWithQuery(query)
		t.NoError(err)

		t.Equal(DefaultGCacheSize, ca.size)
		t.Equal(expire, ca.expire)
	}
}

func TestGCache(t *testing.T) {
	suite.Run(t, new(testGCache))
}
