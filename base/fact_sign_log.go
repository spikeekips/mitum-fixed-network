package base

import "github.com/rs/zerolog"

func MarshalZerologFactSign(fs FactSign) *zerolog.Event {
	e := zerolog.Dict()

	m, ok := fs.(zerolog.LogObjectMarshaler)
	if ok {
		m.MarshalZerologObject(e)

		return e
	}

	return e.Stringer("signer", fs.Signer()).
		Stringer("signature", fs.Signature()).
		Stringer("signed_at", fs.SignedAt())
}
