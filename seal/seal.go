package seal

import (
	"time"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/valuehash"
)

type Signer interface {
	Sign(key.Privatekey /* private key */, []byte /* additional info */) error
}

// Seal is the container of SealBody.
type Seal interface {
	isvalid.IsValider
	hint.Hinter
	valuehash.HashGenerator                    // geneate new hash of seal
	Hash() valuehash.Hash                      // hash of seal
	BodyHash() valuehash.Hash                  // hash of seal body
	GenerateBodyHash() (valuehash.Hash, error) // geneate new hash of seal body
	Signer() key.Publickey                     // signer's PublicKey
	Signature() key.Signature                  // Signature, signed by key
	SignedAt() time.Time                       // signed(or created) time
}
