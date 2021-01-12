package ballot

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
)

func marshalBallotLog(ballot Ballot, key string, e logging.Emitter, verbose bool) logging.Emitter {
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

func (ib INITBallotV0) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	return marshalBallotLog(ib, key, e, verbose)
}

func (pr ProposalV0) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	return marshalBallotLog(pr, key, e, verbose)
}

func (sb SIGNBallotV0) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	return marshalBallotLog(sb, key, e, verbose)
}

func (ab ACCEPTBallotV0) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	return marshalBallotLog(ab, key, e, verbose)
}
