package mitum

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
	Hash() valuehash.Hash                            // hash of seal
	GenerateHash([]byte) (valuehash.Hash, error)     // geneate new hash of seal
	BodyHash() valuehash.Hash                        // hash of seal body
	GenerateBodyHash([]byte) (valuehash.Hash, error) // geneate new hash of seal body
	Signer() key.Publickey                           // signer's PublicKey
	Signature() key.Signature                        // Signature, signed by key
	SignedAt() time.Time                             // signed(or created) time
}
