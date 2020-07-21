package hint

import (
	"github.com/spikeekips/mitum/util/logging"
)

func (ty Type) MarshalLog(key string, e logging.Emitter, _ bool) logging.Emitter {
	return e.Str(key, ty.Verbose())
}
