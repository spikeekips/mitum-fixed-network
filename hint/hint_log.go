package hint

import "github.com/spikeekips/mitum/logging"

func (ht Hint) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	if !verbose {
		return e.Str(key, ht.Verbose())
	}

	return e.Dict(key, logging.Dict().
		HintedVerbose("type", ht.Type(), true).
		Str("version", ht.Version().String()),
	)
}
