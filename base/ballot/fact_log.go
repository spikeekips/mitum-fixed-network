package ballot

import (
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
)

func marshalZerologFact(fact base.BallotFact) *zerolog.Event {
	if fact == nil {
		return nil
	}

	e := zerolog.Dict().
		Stringer("hash", fact.Hash()).
		Stringer("stage", fact.Stage()).
		Int64("height", fact.Height().Int64()).
		Uint64("round", fact.Round().Uint64())

	switch t := fact.(type) {
	case INITFact:
		e = e.Stringer("previous_block", t.previousBlock)
	case ProposalFact:
		seals := t.Seals()
		sls := make([]string, len(seals))
		for i := range seals {
			if seals[i] == nil {
				continue
			}

			sls[i] = seals[i].String()
		}

		e = e.Strs("seals", sls)
	case ACCEPTFact:
		e = e.Stringer("proposal", t.Proposal()).
			Stringer("new_block", t.NewBlock())
	}

	return e
}
