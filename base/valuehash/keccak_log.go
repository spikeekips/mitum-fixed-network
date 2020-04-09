package valuehash

import (
	"github.com/spikeekips/mitum/util/logging"
)

func (s256 SHA256) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	return marshalLog(s256, key, e, verbose)
}

func (s512 SHA512) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	return marshalLog(s512, key, e, verbose)
}
