package base

import "github.com/rs/zerolog"

func (fs BaseBallotFactSign) MarshalZerologObject(e *zerolog.Event) {
	e.
		Stringer("node", fs.Node()).
		Stringer("signer", fs.Signer()).
		Stringer("signature", fs.Signature()).
		Stringer("signed_at", fs.SignedAt())
}
