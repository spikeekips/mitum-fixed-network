package hint

import (
	"io"

	"github.com/ethereum/go-ethereum/rlp"
)

type hintRLP struct {
	T Type
	V Version
}

func (ht Hint) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, hintRLP{T: ht.t, V: ht.version})
}

func (ht *Hint) DecodeRLP(stream *rlp.Stream) error {
	var h hintRLP
	if err := stream.Decode(&h); err != nil {
		return err
	}

	ht.t = h.T
	ht.version = h.V

	return nil
}
