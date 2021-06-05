package deploy

import (
	"net/http"

	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/encoder"
)

func NewDeployKeyNewHandler(ks *DeployKeyStorage, enc encoder.Encoder) network.HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if i, err := ks.New(); err != nil {
			network.WriteProblemWithError(w, http.StatusInternalServerError, err)

			return
		} else if j, err := enc.Marshal(i); err != nil {
			network.WriteProblemWithError(w, http.StatusInternalServerError, err)

			return
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write(j)
		}
	}
}
