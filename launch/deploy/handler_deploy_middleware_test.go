package deploy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/network"
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

	pr, err := network.LoadProblemFromResponse(res)
	t.NoError(err)
	t.Contains(pr.Title(), "empty token")
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

	pr, err := network.LoadProblemFromResponse(res)
	t.NoError(err)
	t.Contains(pr.Title(), "empty signature")
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

	pr, err := network.LoadProblemFromResponse(res)
	t.NoError(err)
	t.Contains(pr.Title(), "failed to verify token and signature")
}

func TestDeployKeyByTokenMiddleware(t *testing.T) {
	suite.Run(t, new(testDeployKeyByTokenMiddleware))
}

type testDeployByKeyMiddleware struct {
	suite.Suite
	ks *DeployKeyStorage
}

func (t *testDeployByKeyMiddleware) SetupTest() {
	t.ks, _ = NewDeployKeyStorage(nil)
}

func (t *testDeployByKeyMiddleware) TestWithoutKey() {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	mw := NewDeployByKeyMiddleware(t.ks)
	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))

	handler.ServeHTTP(w, r)
	res := w.Result()
	t.Equal(http.StatusUnauthorized, res.StatusCode)
	t.NotEmpty(res.Header.Get("WWW-Authenticate"))
}

func (t *testDeployByKeyMiddleware) TestUnknownKey() {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", util.UUID().String())

	mw := NewDeployByKeyMiddleware(t.ks)
	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))

	handler.ServeHTTP(w, r)
	res := w.Result()
	t.Equal(http.StatusForbidden, res.StatusCode)

	pr, err := network.LoadProblemFromResponse(res)
	t.NoError(err)
	t.Contains(pr.Title(), "unknown deploy key")
}

func (t *testDeployByKeyMiddleware) TestValidKey() {
	dk, err := t.ks.New()
	t.NoError(err)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", dk.Key())

	mw := NewDeployByKeyMiddleware(t.ks)

	var requested bool
	handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = true
	}))

	handler.ServeHTTP(w, r)
	res := w.Result()
	t.Equal(http.StatusOK, res.StatusCode)

	t.True(requested)
}

func TestDeployByKeyMiddleware(t *testing.T) {
	suite.Run(t, new(testDeployByKeyMiddleware))
}
