package ballot

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
)

func marshalLog(ballot Ballot, key string, e logging.Emitter, verbose bool) logging.Emitter {
	if !verbose {
		return e.Dict(key, logging.Dict().
			Hinted("hash", ballot.Hash()).
			Hinted("stage", ballot.Stage()).
			Hinted("node", ballot.Node()).
			Hinted("height", ballot.Height()).
			Hinted("round", ballot.Round()),
		)
	}

	r, _ := jsonenc.Marshal(ballot)

	return e.RawJSON(key, r)
}

func (ib INITV0) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	return marshalLog(ib, key, e, verbose)
}

func (pr ProposalV0) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	return marshalLog(pr, key, e, verbose)
}

func (sb SIGNV0) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	return marshalLog(sb, key, e, verbose)
}

func (ab ACCEPTV0) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	return marshalLog(ab, key, e, verbose)
}
