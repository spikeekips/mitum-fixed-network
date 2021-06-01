package deploy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/suite"
)

type testDeployKeyRevokeHandler struct {
	baseDeployKeyHandler
}

func (t *testDeployKeyRevokeHandler) TestNew() {
	ks, err := NewDeployKeyStorage(nil)
	t.NoError(err)

	dks := map[string]DeployKey{}
	for i := 0; i < 3; i++ {
		dk, err := ks.New()
		t.NoError(err)
		dks[dk.Key()] = dk
	}

	handler := NewDeployKeyRevokeHandler(ks)
	router := mux.NewRouter()
	router.Name("deploy-key-key").Path(QuicHandlerPathDeployKeyKeySuffix).Handler(http.HandlerFunc(handler))

	for i := range dks {
		k := dks[i]

		w := httptest.NewRecorder()
		r := httptest.NewRequest("DELETE", "/"+k.Key(), nil)

		router.ServeHTTP(w, r)

		res := w.Result()
		t.Equal(http.StatusOK, res.StatusCode)

		t.False(ks.Exists(k.Key()))
	}

	t.Equal(0, ks.Len())
}

func TestDeployKeyRevokeHandler(t *testing.T) {
	suite.Run(t, new(testDeployKeyRevokeHandler))
}
