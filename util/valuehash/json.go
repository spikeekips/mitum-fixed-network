package valuehash

import (
	"github.com/spikeekips/mitum/util"
)

func marshalJSON(h Hash) ([]byte, error) {
	return util.JSON.Marshal(h.String())
}
