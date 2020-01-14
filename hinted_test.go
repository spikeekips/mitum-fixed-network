package mitum

/*
import (
	"encoding/json"
	"testing"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/stretchr/testify/suite"
)

type something interface {
	something() string
}

type somethingHintedA struct {
	A string
}

func (sh somethingHintedA) Hint() hint.Hint {
	h, err := hint.NewHint(
		hint.Type([2]byte{0xff, 0x20}),
		"0.1",
	)
	if err != nil {
		panic(err)
	}

	return h
}

func (sh somethingHintedA) something() string {
	return sh.A
}

func (sh somethingHintedA) MarshalJSON() ([]byte, error) {
	return (encoder.JSON{}).Encode(map[string]interface{}{
		"_hint": sh.Hint(),
		"A":     sh.A,
	})
}

type somethingHintedB struct {
	S something
	B string
}

func (sh somethingHintedB) Hint() hint.Hint {
	h, err := hint.NewHint(
		hint.Type([2]byte{0xff, 0x21}),
		"0.1",
	)
	if err != nil {
		panic(err)
	}

	return h
}

func (sh somethingHintedB) something() string {
	return sh.B
}

func (sh somethingHintedB) MarshalJSON() ([]byte, error) {
	return (encoder.JSON{}).Encode(map[string]interface{}{
		"_hint": sh.Hint(),
		"S":     sh.S,
		"B":     sh.B,
	})
}

func (sh *somethingHintedB) DecodeJSON(enc encoder.Encoder, hintset *Hintset, b []byte) error {
	var m struct {
		B string
		S json.RawMessage
	}
	if err := enc.Decode(b, &m); err != nil {
		return err
	}

	if s, err := hintset.DecodeHinter(enc, m.S, nil); err != nil {
		return err
	} else {
		sh.S = s.(something)
	}

	sh.B = m.B

	return nil
}

type somethingHinted struct {
	S something
	C string
}

func (sh somethingHinted) Hint() hint.Hint {
	h, err := hint.NewHint(
		hint.Type([2]byte{0xff, 0x22}),
		"0.1",
	)
	if err != nil {
		panic(err)
	}

	return h
}

func (sh somethingHinted) MarshalJSON() ([]byte, error) {
	return (encoder.JSON{}).Encode(map[string]interface{}{
		"_hint": sh.Hint(),
		"S":     sh.S,
		"C":     sh.C,
	})
}

func (sh *somethingHinted) DecodeJSON(enc encoder.Encoder, hintset *Hintset, b []byte) error {
	var m struct {
		C string
		S json.RawMessage
	}
	if err := enc.Decode(b, &m); err != nil {
		return err
	}

	if s, err := hintset.DecodeHinter(enc, m.S, nil); err != nil {
		return err
	} else {
		sh.S = s.(something)
	}

	sh.C = m.C

	return nil
}

type somethingUnhinted struct {
	S something
	C string
}

func (sh *somethingUnhinted) DecodeJSON(enc encoder.Encoder, hintset *Hintset, b []byte) error {
	var m struct {
		C string
		S json.RawMessage
	}
	if err := enc.Decode(b, &m); err != nil {
		return err
	}

	if s, err := hintset.DecodeHinter(enc, m.S, nil); err != nil {
		return err
	} else {
		sh.S = s.(something)
	}

	sh.C = m.C

	return nil
}

type testHinted struct {
	suite.Suite
	encs *encoder.Encoders
}

func (t *testHinted) SetupSuite() {
	_ = hint.RegisterType(somethingHintedA{}.Hint().Type(), "something-hinted-a")
	_ = hint.RegisterType(somethingHintedB{}.Hint().Type(), "something-hinted-b")
	_ = hint.RegisterType(somethingHinted{}.Hint().Type(), "something-hinted")

	_ = hint.RegisterType((encoder.JSON{}).Hint().Type(), "json-encoder")
	t.encs = encoder.NewEncoders()
	t.NoError(t.encs.Add(encoder.JSON{}))
}

func (t *testHinted) TestNew() {
	hs := NewHintset()
	t.NoError(hs.Add(somethingHinted{}))
}

func (t *testHinted) TestMarshalJSONHinted() {
	hs := NewHintset()
	t.NoError(hs.Add(somethingHintedA{}))
	t.NoError(hs.Add(somethingHinted{}))

	c := somethingHinted{
		S: somethingHintedA{A: "A"},
		C: "C",
	}

	b, err := (encoder.JSON{}).Encode(c)
	t.NoError(err)

	// unmarshal
	var unmarshaled somethingHinted
	_, err = hs.DecodeHinter(encoder.JSON{}, b, &unmarshaled)
	t.NoError(err)

	t.True(c.Hint().Equal(unmarshaled.Hint()))
	t.Equal(c.C, unmarshaled.C)
	t.Implements((*something)(nil), unmarshaled.S)
	t.Equal("A", unmarshaled.S.something())
}

func (t *testHinted) TestMarshalJSONUninted() {
	hs := NewHintset()
	t.NoError(hs.Add(somethingHintedA{}))

	c := somethingUnhinted{
		S: somethingHintedA{A: "A"},
		C: "C",
	}

	b, err := (encoder.JSON{}).Encode(c)
	t.NoError(err)

	// unmarshal
	var uc somethingUnhinted
	err = hs.Decode(encoder.JSON{}, b, &uc)
	t.NoError(err)

	t.Equal(c.C, uc.C)
	t.Implements((*something)(nil), uc.S)
	t.Equal("A", uc.S.something())
}

func (t *testHinted) TestMarshalJSONNested() {
	hs := NewHintset()
	t.NoError(hs.Add(somethingHintedA{}))
	t.NoError(hs.Add(somethingHintedB{}))

	c := somethingUnhinted{
		S: somethingHintedB{
			S: somethingHintedA{
				A: "AA",
			},
			B: "B",
		},
		C: "C",
	}

	b, err := (encoder.JSON{}).Encode(c)
	t.NoError(err)

	// unmarshal
	var uc somethingUnhinted
	err = hs.Decode(encoder.JSON{}, b, &uc)
	t.NoError(err)

	t.Equal(c.C, uc.C)
	t.Implements((*something)(nil), uc.S)
	t.Equal("C", uc.C)
	t.Equal("B", uc.S.something())

	t.IsType(&somethingHintedB{}, uc.S)
	t.Equal("AA", uc.S.(*somethingHintedB).S.something())
	t.IsType(&somethingHintedA{}, uc.S.(*somethingHintedB).S)
}

func TestHinted(t *testing.T) {
	suite.Run(t, new(testHinted))
}
*/
