package deploy

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testDeployKeyTokenHandler struct {
	baseDeployKeyHandler
}

func (t *testDeployKeyTokenHandler) TestNew() {
	handler := NewDeployKeyTokenHandler(t.cache, time.Second*3)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, nil)

	res := w.Result()
	t.Equal(http.StatusOK, res.StatusCode)
	t.Equal("application/json", res.Header.Get("content-type"))

	b, err := ioutil.ReadAll(res.Body)
	t.NoError(err)

	var um map[string]string
	t.NoError(jsonenc.Unmarshal(b, &um))

	t.NotEmpty(um["token"])
	t.Equal("3s", um["expired"])
}

func (t *testDeployKeyTokenHandler) TestVerifyTokenSignature() {
	lk := key.MustNewBTCPrivatekey()
	token := util.UUID().String()
	networkID := util.UUID().Bytes()

	t.NoError(t.cache.Set(token, nil, 0))

	sig, err := DeployKeyTokenSignature(lk, token, networkID)
	t.NoError(err)

	t.NoError(VerifyDeployKeyToken(t.cache, lk.Publickey(), token, networkID, sig))
}

func (t *testDeployKeyTokenHandler) TestVerifyTokenSignatureBadToken() {
	lk := key.MustNewBTCPrivatekey()
	token := util.UUID().String()
	networkID := util.UUID().Bytes()

	t.NoError(t.cache.Set(token, nil, 0))

	sig, err := DeployKeyTokenSignature(lk, token, networkID)
	t.NoError(err)

	unknownToken := util.UUID().String()
	t.NoError(t.cache.Set(unknownToken, nil, 0))

	err = VerifyDeployKeyToken(t.cache, lk.Publickey(), unknownToken, networkID, sig)
	t.True(xerrors.Is(err, key.SignatureVerificationFailedError))
}

func TestDeployKeyTokenHandler(t *testing.T) {
	suite.Run(t, new(testDeployKeyTokenHandler))
}
