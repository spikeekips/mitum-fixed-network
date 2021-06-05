package deploy

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
)

type testDeployKeyKeysHandler struct {
	baseDeployKeyHandler
}

func (t *testDeployKeyKeysHandler) TestNew() {
	ks, err := NewDeployKeyStorage(nil)
	t.NoError(err)

	dks := map[string]DeployKey{}
	for i := 0; i < 3; i++ {
		dk, err := ks.New()
		t.NoError(err)
		dks[dk.Key()] = dk
	}

	handler := NewDeployKeyKeysHandler(ks, t.enc)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	handler(w, r)

	res := w.Result()
	t.Equal(http.StatusOK, res.StatusCode)

	b, err := ioutil.ReadAll(res.Body)
	t.NoError(err)

	var l []json.RawMessage
	t.NoError(t.enc.Unmarshal(b, &l))
	t.Equal(3, len(l))

	var udk []DeployKey
	for i := range l {
		var dk DeployKey
		t.NoError(t.enc.Unmarshal(l[i], &dk))
		udk = append(udk, dk)
	}

	for i := range udk {
		k := udk[i]
		_, found := dks[k.Key()]
		t.True(found)
	}
}

func (t *testDeployKeyKeysHandler) TestEmpty() {
	ks, err := NewDeployKeyStorage(nil)
	t.NoError(err)

	handler := NewDeployKeyKeysHandler(ks, t.enc)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	handler(w, r)

	res := w.Result()
	t.Equal(http.StatusOK, res.StatusCode)

	b, err := ioutil.ReadAll(res.Body)
	t.NoError(err)

	var l []json.RawMessage
	t.NoError(t.enc.Unmarshal(b, &l))
	t.Equal(0, len(l))
}

func TestDeployKeyKeysHandler(t *testing.T) {
	suite.Run(t, new(testDeployKeyKeysHandler))
}
