package key

import (
	"github.com/spikeekips/mitum/util"
)

func marshalJSONStringKey(k Key) ([]byte, error) {
	return util.JSON.Marshal(k.String())
}
