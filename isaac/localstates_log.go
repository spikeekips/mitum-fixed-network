package isaac

import (
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util/logging"
)

func (ls *Localstate) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	var manifest block.Manifest
	if m, err := ls.Storage().LastManifest(); err != nil {
		return e
	} else {
		manifest = m
	}

	return e.Dict(key, logging.Dict().
		HintedVerbose("last_block", manifest, verbose),
	)
}
