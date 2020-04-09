package base

import "github.com/spikeekips/mitum/util/logging"

func (st Stage) MarshalLog(key string, e logging.Emitter, _ bool) logging.Emitter {
	return e.Str(key, st.String())
}
