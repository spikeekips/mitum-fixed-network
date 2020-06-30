package valuehash

import (
	"github.com/spikeekips/mitum/util/logging"
)

func (hs SHA256) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	return marshalLog(hs, key, e, verbose)
}

func (hs SHA512) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	return marshalLog(hs, key, e, verbose)
}
