package isaac

import "github.com/spikeekips/mitum/logging"

func (st Stage) MarshalLog(key string, e logging.Emitter, _ bool) logging.Emitter {
	return e.Str(key, st.String())
}
