package mitum

import (
	"time"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/valuehash"
)

// Seal is the container of SealBody.
type Seal interface {
	hint.Hinter
	Hash() valuehash.Hash     // hash of seal
	Signer() Address          // signer's PublicKey
	Signature() key.Signature // Signature, signed by key
	SignedAt() time.Time      // signed(or created) time
	Body() SealBody           // SealBody
}

// SealBody is the body of seal, it contains main information of seal.
type SealBody interface {
	hint.Hinter
	Hash() valuehash.Hash // hash of SealBody
}
