package isaac

import "github.com/spikeekips/mitum/logging"

func (ls *Localstate) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	lastBlock := ls.LastBlock()
	if lastBlock == nil {
		return e
	}

	return e.Dict(key, logging.Dict().
		HintedVerbose("last_block", lastBlock, verbose),
	)
}
