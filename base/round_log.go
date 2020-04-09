package base

import "github.com/spikeekips/mitum/util/logging"

func (rn Round) MarshalLog(key string, e logging.Emitter, _ bool) logging.Emitter {
	return e.Uint64(key, rn.Uint64())
}
