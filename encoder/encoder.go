package encoder

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
)

type Encoder interface {
	hint.Hinter
	Marshal(interface{}) ([]byte, error)
	Unmarshal([]byte, interface{}) error
	Encoder(interface{}) (func(*HintEncoder) (interface{}, error), bool)
	Decoder(interface{}) (func(*HintEncoder, []byte) error, bool)
	EncodeHint(hinter hint.Hinter, encoded interface{}) ([]byte, error)
	DecodeHint(b []byte) (hint.Hint, error)
}

func CheckEncodedHint(enc Encoder, h hint.Hint, b []byte) error {
	if he, err := enc.DecodeHint(b); err != nil {
		return err
	} else if h != he {
		return xerrors.Errorf("hint does not match: %v != %v", h.Verbose(), he.Verbose())
	}

	return nil
}
