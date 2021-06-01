package deploy

import (
	"net/http"
	"strings"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/logging"
)

type DeployKeyByTokenMiddleware struct {
	*logging.Logging
	cache     cache.Cache
	localKey  key.Publickey
	networkID base.NetworkID
}

func NewDeployKeyByTokenMiddleware(
	c cache.Cache,
	localKey key.Publickey,
	networkID base.NetworkID,
) *DeployKeyByTokenMiddleware {
	return &DeployKeyByTokenMiddleware{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "deploy-key-token-middleware")
		}),
		cache:     c,
		localKey:  localKey,
		networkID: networkID,
	}
}

func (md *DeployKeyByTokenMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// NOTE check token and signature
		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if len(token) < 1 {
			network.HTTPError(w, http.StatusUnauthorized)
			return
		}

		var sig key.Signature
		if i := strings.TrimSpace(r.URL.Query().Get("signature")); len(i) < 1 {
			network.HTTPError(w, http.StatusUnauthorized)
			return
		} else {
			sig = key.NewSignatureFromString(i)
		}

		if err := VerifyDeployKeyToken(md.cache, md.localKey, token, md.networkID, []byte(sig)); err != nil {
			md.Log().Error().Err(err).Msg("failed to verify token and signature")
			network.HTTPError(w, http.StatusUnauthorized)

			return
		}

		// NOTE expire token
		_ = md.cache.Set(token, nil, time.Nanosecond)

		next.ServeHTTP(w, r)
	})
}
