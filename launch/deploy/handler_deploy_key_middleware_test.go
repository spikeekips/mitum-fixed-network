package deploy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/stretchr/testify/suite"
)

type testDeployKeyByTokenMiddleware struct {
	suite.Suite
	cache     cache.Cache
	networkID base.NetworkID
	handler   http.Handler
}

func (t *testDeployKeyByTokenMiddleware) SetupTest() {
	c, err := cache.NewGCache("lru", 100*100, time.Minute)
	t.NoError(err)

	t.cache = c
	t.networkID = base.NetworkID(util.UUID().Bytes())
	t.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}

func (t *testDeployKeyByTokenMiddleware) TestNew() {
	lk := key.MustNewBTCPrivatekey()
	md := NewDeployKeyByTokenMiddleware(t.cache, lk.Publickey(), t.networkID)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	token := util.UUID().String()

	sig, err := DeployKeyTokenSignature(lk, token, t.networkID)
	t.NoError(err)

	query := r.URL.Query()
	query.Set("token", token)
	query.Set("signature", sig.String())

	r.URL.RawQuery = query.Encode()

	t.NoError(t.cache.Set(token, nil, 0))

	var served bool
	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served = true
	})
	md.Middleware(emptyHandler).ServeHTTP(w, r)

	res := w.Result()
	t.Equal(http.StatusOK, res.StatusCode)
	t.True(served)

	// NOTE token will be removed
	t.False(t.cache.Has(token))
}

func (t *testDeployKeyByTokenMiddleware) TestMissingToken() {
	lk := key.MustNewBTCPrivatekey()
	md := NewDeployKeyByTokenMiddleware(t.cache, lk.Publickey(), t.networkID)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	token := util.UUID().String()

	sig, err := DeployKeyTokenSignature(lk, token, t.networkID)
	t.NoError(err)

	query := r.URL.Query()
	query.Set("signature", sig.String())

	r.URL.RawQuery = query.Encode()

	t.NoError(t.cache.Set(token, nil, 0))

	md.Middleware(t.handler).ServeHTTP(w, r)

	res := w.Result()
	t.Equal(http.StatusUnauthorized, res.StatusCode)
}

func (t *testDeployKeyByTokenMiddleware) TestMissingSignature() {
	lk := key.MustNewBTCPrivatekey()
	md := NewDeployKeyByTokenMiddleware(t.cache, lk.Publickey(), t.networkID)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	token := util.UUID().String()

	query := r.URL.Query()
	query.Set("token", token)

	r.URL.RawQuery = query.Encode()

	t.NoError(t.cache.Set(token, nil, 0))

	md.Middleware(t.handler).ServeHTTP(w, r)

	res := w.Result()
	t.Equal(http.StatusUnauthorized, res.StatusCode)
}

func (t *testDeployKeyByTokenMiddleware) TestUnknownToken() {
	lk := key.MustNewBTCPrivatekey()
	md := NewDeployKeyByTokenMiddleware(t.cache, lk.Publickey(), t.networkID)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	token := util.UUID().String()

	sig, err := DeployKeyTokenSignature(lk, token, t.networkID)
	t.NoError(err)

	query := r.URL.Query()
	query.Set("token", token)
	query.Set("signature", sig.String())

	r.URL.RawQuery = query.Encode()

	md.Middleware(t.handler).ServeHTTP(w, r)

	res := w.Result()
	t.Equal(http.StatusUnauthorized, res.StatusCode)
}

func TestDeployKeyByTokenMiddleware(t *testing.T) {
	suite.Run(t, new(testDeployKeyByTokenMiddleware))
}
