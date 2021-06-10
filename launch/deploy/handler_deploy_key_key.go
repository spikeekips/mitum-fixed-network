package deploy

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/encoder"
	"golang.org/x/xerrors"
)

var QuicHandlerPathDeployKeyKeySuffix = "/{deploy_key:.*}"

func NewDeployKeyKeyHandler(ks *DeployKeyStorage, enc encoder.Encoder) network.HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deployKey, err := loadDeployKeyFromRequestPath(r)
		if err != nil {
			network.WriteProblemWithError(w, http.StatusBadRequest, err)

			return
		}

		if i, found := ks.Key(deployKey); !found {
			network.WriteProblemWithError(w, http.StatusNotFound, xerrors.Errorf("deploy key not found"))

			return
		} else if j, err := enc.Marshal(i); err != nil {
			network.WriteProblemWithError(w, http.StatusInternalServerError, err)

			return
		} else {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(j)
		}
	}
}

func loadDeployKeyFromRequestPath(r *http.Request) (string, error) {
	vars := mux.Vars(r)
	i := strings.TrimSpace(vars["deploy_key"])
	if len(i) < 1 {
		return "", xerrors.Errorf("empty deploy key")
	}
	return i, nil
}
