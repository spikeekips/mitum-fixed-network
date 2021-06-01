package deploy

/*
import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spikeekips/mitum/base/key"
	"github.com/stretchr/testify/suite"
)

type testDeployKeyKeysHandler struct {
	baseDeployKeyHandler
}

func (t *testDeployKeyKeysHandler) TestNew() {
	ks, err := NewKeys(nil)
	t.NoError(err)

	dks := map[string]key.Publickey{}
	for i := 0; i < 3; i++ {
		dk := key.MustNewBTCPrivatekey().Publickey()
		t.NoError(ks.Add(dk))
		dks[dk.String()] = dk
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
		j, err := t.enc.DecodeByHint(l[i])
		t.NoError(err)
		c := j.(DeployKey)
		udk = append(udk, c)
	}

	for i := range udk {
		k := udk[i]
		_, found := dks[k.Key().String()]
		t.True(found)
	}
}

func (t *testDeployKeyKeysHandler) TestEmpty() {
	ks, err := NewKeys(nil)
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
*/
