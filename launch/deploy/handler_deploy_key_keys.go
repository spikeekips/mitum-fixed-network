package deploy

import (
	"net/http"

	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/encoder"
)

func NewDeployKeyKeysHandler(ks *DeployKeyStorage, enc encoder.Encoder) network.HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := make([]DeployKey, ks.Len())

		var i int
		ks.Traverse(func(k DeployKey) bool {
			m[i] = k

			i++

			return true
		})

		if i, err := enc.Marshal(m); err != nil {
			network.HTTPError(w, http.StatusInternalServerError)

			return
		} else {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(i)
		}
	}
}
