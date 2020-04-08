package isaac

import "github.com/spikeekips/mitum/logging"

func (rn Round) MarshalLog(key string, e logging.Emitter, _ bool) logging.Emitter {
	return e.Uint64(key, rn.Uint64())
}
