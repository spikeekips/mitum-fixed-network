// +build test

package encoder

import "github.com/spikeekips/mitum/util/hint"

// TestAddHinter add Hinter with it's Type if not yet added.
func (es *Encoders) TestAddHinter(ht hint.Hinter) error {
	if !es.hintset.HasType(ht.Hint().Type()) {
		if err := es.hintset.AddType(ht.Hint().Type()); err != nil {
			return err
		}
	}

	return es.AddHinter(ht)
}
