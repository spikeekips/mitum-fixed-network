package hint

import (
	"fmt"

	"github.com/spikeekips/mitum/util/logging"
)

func (ty Type) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	if !verbose {
		return e.Str(key, ty.Verbose())
	}

	name := ty.String()
	if len(name) > 0 {
		name = fmt.Sprintf("(%s)", name)
	}

	return e.Dict(key, logging.Dict().
		Str("code", fmt.Sprintf("%x", [2]byte(ty))).
		Str("name", name),
	)
}
