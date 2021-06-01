package deploy

import (
	"net/http"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
)

type DeployKeyTokenHandler struct {
	*logging.Logging
	cache   cache.Cache
	expired time.Duration
}

func NewDeployKeyTokenHandler(c cache.Cache, expired time.Duration) *DeployKeyTokenHandler {
	return &DeployKeyTokenHandler{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "handler-deploy-key-token")
		}),
		cache:   c,
		expired: expired,
	}
}

func (hn *DeployKeyTokenHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	token := "t-" + util.UUID().String()

	if i, err := jsonenc.Marshal(map[string]interface{}{
		"token":   token,
		"expired": hn.expired.String(),
	}); err != nil {
		hn.Log().Error().Err(err).Msg("failed to marshal token output")
		network.HTTPError(w, http.StatusInternalServerError)

		return
	} else if err := hn.cache.Set(token, nil, hn.expired); err != nil {
		hn.Log().Error().Err(err).Msg("failed to set token in cache")
		network.HTTPError(w, http.StatusInternalServerError)

		return
	} else {
		w.Header().Set("Content-Type", "application/json")

		_, _ = w.Write(i)
	}
}

func DeployKeyTokenSignature(localKey key.Privatekey, token string, networkID base.NetworkID) (key.Signature, error) {
	return localKey.Sign(util.ConcatBytesSlice([]byte(token), networkID))
}

func VerifyDeployKeyToken(
	c cache.Cache,
	localKey key.Publickey,
	token string,
	networkID base.NetworkID,
	signature key.Signature,
) error {
	if !c.Has(token) {
		return util.NotFoundError.Errorf("unknown token")
	}

	return localKey.Verify(util.ConcatBytesSlice(
		[]byte(token),
		networkID,
	), signature)
}
