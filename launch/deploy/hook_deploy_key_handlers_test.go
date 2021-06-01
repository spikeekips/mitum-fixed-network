package deploy

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
)

type testDeployKeyHandlers struct {
	baseDeployKeyHandler
	isaac.StorageSupportTest
	local  *network.LocalNode
	policy *isaac.LocalPolicy
	db     storage.Database
	dks    *DeployKeyStorage
}

func (t *testDeployKeyHandlers) SetupSuite() {
	t.StorageSupportTest.SetupSuite()

	t.NoError(t.Encs.AddHinter(key.BTCPublickeyHinter))
}

func (t *testDeployKeyHandlers) handlers(router *mux.Router) *deployKeyHandlers {
	ctx := context.WithValue(context.Background(), config.ContextValueLog, log)

	ctx = context.WithValue(ctx, config.ContextValueJSONEncoder, t.JSONEnc)
	t.local = network.RandomLocalNode("local", nil)
	ctx = context.WithValue(ctx, process.ContextValueLocalNode, t.local)
	t.policy = isaac.NewLocalPolicy(util.UUID().Bytes())
	ctx = context.WithValue(ctx, process.ContextValuePolicy, t.policy)

	t.db = t.Database(t.Encs, nil)
	ctx = context.WithValue(ctx, process.ContextValueDatabase, t.db)

	t.dks, _ = NewDeployKeyStorage(t.db)
	ctx = context.WithValue(ctx, ContextValueDeployKeyStorage, t.dks)

	handlers, err := newDeployKeyHandlers(ctx, func(prefix string) *mux.Route {
		return router.Name(prefix).Path(prefix)
	})
	t.NoError(err)

	t.NoError(handlers.setHandlers())

	return handlers
}

func (t *testDeployKeyHandlers) token(router *mux.Router) string {
	// NOTE get token
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", QuicHandlerPathDeployKeyToken, nil)
	router.ServeHTTP(w, r)

	res := w.Result()
	t.Equal(http.StatusOK, res.StatusCode)

	b, err := ioutil.ReadAll(res.Body)
	t.NoError(err)

	var um map[string]string
	t.NoError(jsonenc.Unmarshal(b, &um))

	t.NotEmpty(um["token"])
	t.Equal("30s", um["expired"])

	return um["token"]
}

func (t *testDeployKeyHandlers) tokenAndSignature(router *mux.Router) (string, key.Signature) {
	token := t.token(router)

	sig, err := DeployKeyTokenSignature(t.local.Privatekey(), token, t.policy.NetworkID())
	t.NoError(err)

	return token, sig
}

func (t *testDeployKeyHandlers) TestToken() {
	router := mux.NewRouter()
	_ = t.handlers(router)

	// NOTE get token
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", QuicHandlerPathDeployKeyToken, nil)
	router.ServeHTTP(w, r)

	res := w.Result()
	t.Equal(http.StatusOK, res.StatusCode)

	b, err := ioutil.ReadAll(res.Body)
	t.NoError(err)

	var um map[string]string
	t.NoError(jsonenc.Unmarshal(b, &um))

	t.NotEmpty(um["token"])
	t.Equal("30s", um["expired"])
}

func (t *testDeployKeyHandlers) TestNewKeyBadToken() {
	router := mux.NewRouter()
	_ = t.handlers(router)

	token := t.token(router)

	sig, err := DeployKeyTokenSignature(t.local.Privatekey(), token, t.policy.NetworkID())
	t.NoError(err)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", QuicHandlerPathDeployKeyNew, nil)

	query := r.URL.Query()
	query.Set("token", token+"0")
	query.Set("signature", sig.String())
	r.URL.RawQuery = query.Encode()
	router.ServeHTTP(w, r)

	res := w.Result()
	t.Equal(http.StatusUnauthorized, res.StatusCode)
}

func (t *testDeployKeyHandlers) TestNewKeyBadSignature() {
	router := mux.NewRouter()
	_ = t.handlers(router)

	token := t.token(router)

	sig, err := DeployKeyTokenSignature(t.local.Privatekey(), token, t.policy.NetworkID())
	t.NoError(err)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", QuicHandlerPathDeployKeyNew, nil)

	query := r.URL.Query()
	query.Set("token", token)
	query.Set("signature", sig.String()+"0")
	r.URL.RawQuery = query.Encode()
	router.ServeHTTP(w, r)

	res := w.Result()
	t.Equal(http.StatusUnauthorized, res.StatusCode)
}

func (t *testDeployKeyHandlers) TestNewKey() {
	router := mux.NewRouter()
	_ = t.handlers(router)

	token, sig := t.tokenAndSignature(router)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", QuicHandlerPathDeployKeyNew, nil)

	query := r.URL.Query()
	query.Set("token", token)
	query.Set("signature", sig.String())
	r.URL.RawQuery = query.Encode()
	router.ServeHTTP(w, r)

	res := w.Result()
	t.Equal(http.StatusCreated, res.StatusCode)

	b, err := ioutil.ReadAll(res.Body)
	t.NoError(err)

	var um map[string]string
	t.NoError(jsonenc.Unmarshal(b, &um))

	t.NotEmpty(um["key"])
	t.NotEmpty(um["added_at"])

	t.True(t.dks.Exists(um["key"]))
}

func (t *testDeployKeyHandlers) newKey(router *mux.Router) string {
	token, sig := t.tokenAndSignature(router)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", QuicHandlerPathDeployKeyNew, nil)

	query := r.URL.Query()
	query.Set("token", token)
	query.Set("signature", sig.String())
	r.URL.RawQuery = query.Encode()
	router.ServeHTTP(w, r)

	res := w.Result()
	t.Equal(http.StatusCreated, res.StatusCode)

	b, err := ioutil.ReadAll(res.Body)
	t.NoError(err)

	var um map[string]string
	t.NoError(jsonenc.Unmarshal(b, &um))

	t.NotEmpty(um["key"])
	t.NotEmpty(um["added_at"])

	return um["key"]
}

func (t *testDeployKeyHandlers) TestKey() {
	router := mux.NewRouter()
	_ = t.handlers(router)

	dk := t.newKey(router)

	token, sig := t.tokenAndSignature(router)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", QuicHandlerPathDeployKeyKeyPrefix+"/"+dk, nil)

	query := r.URL.Query()
	query.Set("token", token)
	query.Set("signature", sig.String())
	r.URL.RawQuery = query.Encode()
	router.ServeHTTP(w, r)

	res := w.Result()
	t.Equal(http.StatusOK, res.StatusCode)

	b, err := ioutil.ReadAll(res.Body)
	t.NoError(err)

	var um map[string]string
	t.NoError(jsonenc.Unmarshal(b, &um))

	t.NotEmpty(um["key"])
	t.NotEmpty(um["added_at"])

	t.Equal(dk, um["key"])
}

func (t *testDeployKeyHandlers) TestKeys() {
	router := mux.NewRouter()
	_ = t.handlers(router)

	for i := 0; i < 3; i++ {
		_, err := t.dks.New()
		t.NoError(err)
	}

	token, sig := t.tokenAndSignature(router)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", QuicHandlerPathDeployKeyKeys, nil)

	query := r.URL.Query()
	query.Set("token", token)
	query.Set("signature", sig.String())
	r.URL.RawQuery = query.Encode()
	router.ServeHTTP(w, r)

	res := w.Result()
	t.Equal(http.StatusOK, res.StatusCode)

	b, err := ioutil.ReadAll(res.Body)
	t.NoError(err)

	var um []DeployKey
	t.NoError(jsonenc.Unmarshal(b, &um))

	for i := range um {
		udk := um[i]

		t.True(t.dks.Exists(udk.Key()))
	}
}

func (t *testDeployKeyHandlers) TestKeyRevoke() {
	router := mux.NewRouter()
	_ = t.handlers(router)

	dk := t.newKey(router)

	token, sig := t.tokenAndSignature(router)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", QuicHandlerPathDeployKeyKeyPrefix+"/"+dk, nil)

	query := r.URL.Query()
	query.Set("token", token)
	query.Set("signature", sig.String())
	r.URL.RawQuery = query.Encode()
	router.ServeHTTP(w, r)

	res := w.Result()
	t.Equal(http.StatusOK, res.StatusCode)

	t.False(t.dks.Exists(dk))
}

func TestDeployKeyHandlers(t *testing.T) {
	suite.Run(t, new(testDeployKeyHandlers))
}
