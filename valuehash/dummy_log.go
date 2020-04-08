package valuehash

import "github.com/spikeekips/mitum/logging"

func (dm Dummy) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	return marshalLog(dm, key, e, verbose)
}
