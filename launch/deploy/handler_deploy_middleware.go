package deploy

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

func UnauthorizedError(w http.ResponseWriter, realm string, err error) {
	// NOTE realm is reserved for deploy key scopes
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`MitumDeployKey realm="%s", charset="utf-8"`, realm))
	network.WriteProblemWithError(w, http.StatusUnauthorized, err)
}

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
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
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
			network.WriteProblemWithError(w, http.StatusUnauthorized, xerrors.Errorf("empty token"))
			return
		}

		i := strings.TrimSpace(r.URL.Query().Get("signature"))
		if len(i) < 1 {
			network.WriteProblemWithError(w, http.StatusUnauthorized, xerrors.Errorf("empty signature"))
			return
		}
		sig := key.NewSignatureFromString(i)

		if err := VerifyDeployKeyToken(md.cache, md.localKey, token, md.networkID, []byte(sig)); err != nil {
			md.Log().Error().Err(err).Msg("failed to verify token and signature")
			network.WriteProblemWithError(w, http.StatusUnauthorized,
				xerrors.Errorf("failed to verify token and signature: %w", err))

			return
		}

		// NOTE expire token
		_ = md.cache.Set(token, nil, time.Nanosecond)

		next.ServeHTTP(w, r)
	})
}

type DeployByKeyMiddleware struct {
	*logging.Logging
	ks *DeployKeyStorage
}

func NewDeployByKeyMiddleware(ks *DeployKeyStorage) *DeployByKeyMiddleware {
	return &DeployByKeyMiddleware{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "deploy-by-key-middleware")
		}),
		ks: ks,
	}
}

func (md *DeployByKeyMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if len(auth) < 1 {
			UnauthorizedError(w, "", xerrors.Errorf("empty Authorization"))
			return
		}

		if !md.ks.Exists(auth) {
			network.WriteProblemWithError(w, http.StatusForbidden, xerrors.Errorf("unknown deploy key"))
			return
		}

		next.ServeHTTP(w, r)
	})
}
