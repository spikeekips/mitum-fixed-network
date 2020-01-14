package encoder

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/hint"
)

var (
	bsonHint hint.Hint = hint.MustHint(hint.Type([2]byte{0x01, 0x01}), "0.1")
)

func NewBSONHintEncoder() *HintEncoder {
	return NewHintEncoder(BSON{})
}

type BSON struct {
}

func (be BSON) Hint() hint.Hint {
	return bsonHint
}

func (be BSON) Marshal(v interface{}) ([]byte, error) {
	return bson.Marshal(v)
}

func (be BSON) Unmarshal(b []byte, v interface{}) error {
	return bson.Unmarshal(b, v)
}

func (be BSON) Encoder(target interface{}) (func(*HintEncoder) (interface{}, error), bool) {
	fn, ok := target.(BSONEncoder)
	if !ok {
		return nil, false
	}

	return fn.EncodeBSON, true
}

func (be BSON) Decoder(target interface{}) (func(*HintEncoder, []byte) error, bool) {
	return func(enc *HintEncoder, b []byte) error {
		var raw []byte
		if _, ok := target.(hint.Hinter); ok {
			var h struct {
				H hint.Hint `bson:"_hint"`
				R bson.Raw  `bson:"_raw,omitempty"`
			}

			if err := be.Unmarshal(b, &h); err != nil {
				return err
			}
			raw = h.R
		} else {
			raw = b
		}

		if fn, ok := target.(BSONDecoder); ok {
			return fn.DecodeBSON(enc, raw)
		} else {
			return be.Unmarshal(raw, target)
		}
	}, true
}

// Encode encodes input data with Hint information.
func (be BSON) EncodeHint(hinter hint.Hinter, encoded interface{}) ([]byte, error) {
	if hinter == nil {
		return be.Marshal(encoded)
	}

	return be.Marshal(BSONHinterHead{H: hinter.Hint(), R: encoded})
}

func (be BSON) DecodeHint(b []byte) (hint.Hint, error) {
	var h BSONHinterHead
	if err := bson.Unmarshal(b, &h); err != nil {
		return hint.Hint{}, err
	}

	if err := h.H.IsValid(); err != nil {
		return hint.Hint{}, err
	}

	return h.H, nil
}

type BSONHinterHead struct {
	H hint.Hint   `bson:"_hint"`
	R interface{} `bson:"_raw,omitempty"`
}

func NewBSONHinterHead(ht hint.Hint) BSONHinterHead {
	return BSONHinterHead{H: ht}
}

type BSONEncoder interface {
	EncodeBSON(*HintEncoder) (interface{}, error)
}

type BSONDecoder interface {
	DecodeBSON(*HintEncoder, []byte) error
}
