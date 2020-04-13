package block

import "github.com/spikeekips/mitum/util/logging"

func (bm ManifestV0) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	return e.Dict(key, logging.Dict().
		HintedVerbose("hash", bm.Hash(), verbose).
		HintedVerbose("height", bm.Height(), verbose).
		HintedVerbose("round", bm.Round(), verbose),
	)
}
