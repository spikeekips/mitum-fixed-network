package deploy

import (
	"net/http"

	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
)

func NewDeployKeyRevokeHandler(ks *DeployKeyStorage) network.HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var deployKey string
		if i, err := loadDeployKeyFromRequestPath(r); err != nil {
			network.HTTPError(w, http.StatusBadRequest)

			return
		} else {
			deployKey = i
		}

		if err := ks.Revoke(deployKey); err != nil {
			if xerrors.Is(err, util.NotFoundError) {
				network.HTTPError(w, http.StatusNotFound)

				return
			}

			network.HTTPError(w, http.StatusInternalServerError)

			return
		}
	}
}
