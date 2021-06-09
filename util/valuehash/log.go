package valuehash

import (
	"github.com/spikeekips/mitum/util/logging"
)

func marshalLog(h Hash, key string, e logging.Emitter, _ bool) logging.Emitter {
	return e.Str(key, h.String())
}
