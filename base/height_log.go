package base

import "github.com/spikeekips/mitum/util/logging"

func (ht Height) MarshalLog(key string, e logging.Emitter, _ bool) logging.Emitter {
	return e.Int64(key, ht.Int64())
}
