package deploy

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
)

type testDeployKeyNewHandler struct {
	baseDeployKeyHandler
}

func (t *testDeployKeyNewHandler) TestNew() {
	ks, err := NewDeployKeyStorage(nil)
	t.NoError(err)
	handler := NewDeployKeyNewHandler(ks, t.enc)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	handler(w, r)

	res := w.Result()
	t.Equal(http.StatusCreated, res.StatusCode)

	b, err := ioutil.ReadAll(res.Body)
	t.NoError(err)

	var udk DeployKey
	t.NoError(t.enc.Unmarshal(b, &udk))

	t.True(ks.Exists(udk.Key()))
}

func TestDeployKeyNewHandler(t *testing.T) {
	suite.Run(t, new(testDeployKeyNewHandler))
}
