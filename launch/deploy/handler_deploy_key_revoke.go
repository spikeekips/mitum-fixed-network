package deploy

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
)

func NewDeployKeyRevokeHandler(ks *DeployKeyStorage) network.HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deployKey, err := loadDeployKeyFromRequestPath(r)
		if err != nil {
			network.WriteProblemWithError(w, http.StatusBadRequest, err)

			return
		}

		if err := ks.Revoke(deployKey); err != nil {
			if errors.Is(err, util.NotFoundError) {
				network.WriteProblemWithError(w, http.StatusNotFound, err)

				return
			}

			network.WriteProblemWithError(w, http.StatusInternalServerError, err)

			return
		}
	}
}
