package valuehash

import "github.com/spikeekips/mitum/util/logging"

func (hs Bytes) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	return marshalLog(hs, key, e, verbose)
}
