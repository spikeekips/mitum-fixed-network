package deploy

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
)

type testDeployKeyKeyHandler struct {
	baseDeployKeyHandler
}

func (t *testDeployKeyKeyHandler) TestNew() {
	ks, err := NewDeployKeyStorage(nil)
	t.NoError(err)

	dks := map[string]DeployKey{}
	for i := 0; i < 3; i++ {
		dk, err := ks.New()
		t.NoError(err)
		dks[dk.Key()] = dk
	}

	handler := NewDeployKeyKeyHandler(ks, t.enc)
	router := mux.NewRouter()
	router.Name("deploy-key-key").Path(QuicHandlerPathDeployKeyKeySuffix).Handler(http.HandlerFunc(handler))

	for i := range dks {
		k := dks[i]

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/"+k.Key(), nil)

		router.ServeHTTP(w, r)

		res := w.Result()
		t.Equal(http.StatusOK, res.StatusCode)

		b, err := ioutil.ReadAll(res.Body)
		t.NoError(err)

		var udk DeployKey
		t.NoError(t.enc.Unmarshal(b, &udk))

		t.Equal(k.Key(), udk.Key())
	}
}

func (t *testDeployKeyKeyHandler) TestUnknownKey() {
	ks, err := NewDeployKeyStorage(nil)
	t.NoError(err)

	handler := NewDeployKeyKeyHandler(ks, t.enc)
	router := mux.NewRouter()
	router.Name("deploy-key-key").Path(QuicHandlerPathDeployKeyKeySuffix).Handler(http.HandlerFunc(handler))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/"+util.UUID().String(), nil)

	router.ServeHTTP(w, r)

	res := w.Result()
	t.Equal(http.StatusNotFound, res.StatusCode)
}

func TestDeployKeyKeyHandler(t *testing.T) {
	suite.Run(t, new(testDeployKeyKeyHandler))
}
