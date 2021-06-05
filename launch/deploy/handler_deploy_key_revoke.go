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
			network.WriteProblemWithError(w, http.StatusBadRequest, err)

			return
		} else {
			deployKey = i
		}

		if err := ks.Revoke(deployKey); err != nil {
			if xerrors.Is(err, util.NotFoundError) {
				network.WriteProblemWithError(w, http.StatusNotFound, err)

				return
			}

			network.WriteProblemWithError(w, http.StatusInternalServerError, err)

			return
		}
	}
}
