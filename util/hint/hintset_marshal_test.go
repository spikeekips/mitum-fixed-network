package hint

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
)

type hintedMarshalInfo struct {
	hint Hint
	T    reflect.Type
}

func newHintedMarshalInfo(instance interface{}) (hintedMarshalInfo, error) {
	hinted, ok := instance.(Hinter)
	if !ok {
		return hintedMarshalInfo{}, xerrors.Errorf("not Hinter instance: %%", instance)
	}

	return hintedMarshalInfo{
		hint: hinted.Hint(),
		T:    reflect.TypeOf(instance),
	}, nil
}

func (hm hintedMarshalInfo) Hint() Hint {
	return hm.hint
}

func (hm hintedMarshalInfo) unmarshalJSON(b []byte) (interface{}, error) {
	c := reflect.New(hm.T).Interface()
	if err := util.JSON.Unmarshal(b, c); err != nil {
		return nil, err
	}

	return reflect.ValueOf(c).Elem().Interface(), nil
}

type testHintsetMarshal struct {
	suite.Suite
}

func (t *testHintsetMarshal) TestAddRemove() {
	hintset := NewHintset()

	h, err := NewHint(Type{0xff, 0x20}, "0.1.1")
	t.NoError(err)
	t.NoError(registerType(h.Type(), "0xff20-0.1.1"))

	fh := fieldHinted{H: h} // Create primitive instance

	err = hintset.Add(fh) // Add()
	t.NoError(err)

	hinted, err := hintset.Hinter(h.Type(), h.Version()) // Hinter()
	t.NoError(err)
	t.NotNil(hinted)
	t.Equal(h, hinted.Hint())

	err = hintset.Remove(h.Type(), h.Version()) // Remove()
	t.NoError(err)
}

func (t *testHintsetMarshal) TestJSONMarshalFieldHinted() {
	hintset := NewHintset()

	h, err := NewHint(Type{0xff, 0x21}, "0.1.1")
	t.NoError(err)
	t.NoError(registerType(h.Type(), "0xff21-0.1.1"))

	hm, err := newHintedMarshalInfo(fieldHinted{H: h})
	t.NoError(err)

	err = hintset.Add(hm)
	t.NoError(err)

	fh := fieldHinted{H: h, A: 10, B: "showme"}

	b, err := util.JSON.Marshal(fh)
	t.NoError(err)

	// unmarshal
	hint, err := HintFromJSONMarshaled(b)
	t.NoError(err)
	t.Equal(h.Type(), hint.Type())
	t.Equal(h.Version(), hint.Version())

	hinted, err := hintset.Hinter(hint.Type(), hint.Version())
	t.NoError(err)
	t.Equal(h.Type(), hinted.Hint().Type())
	t.Equal(h.Version(), hinted.Hint().Version())

	info := hinted.(hintedMarshalInfo)

	unmarshaled, err := info.unmarshalJSON(b)
	t.NoError(err)

	t.Equal(fh.Hint(), unmarshaled.(Hinter).Hint())
	t.Equal(fh.A, unmarshaled.(fieldHinted).A)
	t.Equal(fh.B, unmarshaled.(fieldHinted).B)
}

func (t *testHintsetMarshal) TestJSONMarshalMethodHinted() {
	hintset := NewHintset()

	hm, err := newHintedMarshalInfo(methodHinted{})
	t.NoError(err)

	_ = registerType(methodHinted{}.Hint().Type(), "0xff11-v0.0.2")

	err = hintset.Add(hm)
	t.NoError(err)

	mh := methodHinted{A: 33, B: "findme"}

	b, err := util.JSON.Marshal(mh)
	t.NoError(err)

	// unmarshal
	hint, err := HintFromJSONMarshaled(b)
	t.NoError(err)
	t.Equal(mh.Hint().Type(), hint.Type())
	t.Equal(mh.Hint().Version(), hint.Version())

	hinted, err := hintset.Hinter(hint.Type(), hint.Version())
	t.NoError(err)
	t.Equal(mh.Hint().Type(), hinted.Hint().Type())
	t.Equal(mh.Hint().Version(), hinted.Hint().Version())

	info := hinted.(hintedMarshalInfo)

	unmarshaled, err := info.unmarshalJSON(b)
	t.NoError(err)

	t.Equal(mh.Hint(), unmarshaled.(Hinter).Hint())
	t.Equal(mh.A, unmarshaled.(methodHinted).A)
	t.Equal(mh.B, unmarshaled.(methodHinted).B)
}

func (t *testHintsetMarshal) TestJSONMarshalCustomMarshalHinted() {
	hintset := NewHintset()

	hm, err := newHintedMarshalInfo(customMarshalHinted{})
	t.NoError(err)

	_ = registerType(customMarshalHinted{}.Hint().Type(), "0xff12-v0.0.3")

	err = hintset.Add(hm)
	t.NoError(err)

	mh := customMarshalHinted{A: 33, B: "findme"}

	b, err := util.JSON.Marshal(mh)
	t.NoError(err)

	// unmarshal
	hint, err := HintFromJSONMarshaled(b)
	t.NoError(err)
	t.Equal(mh.Hint().Type(), hint.Type())
	t.Equal(mh.Hint().Version(), hint.Version())

	hinted, err := hintset.Hinter(hint.Type(), hint.Version())
	t.NoError(err)
	t.Equal(mh.Hint().Type(), hinted.Hint().Type())
	t.Equal(mh.Hint().Version(), hinted.Hint().Version())

	info := hinted.(hintedMarshalInfo)

	unmarshaled, err := info.unmarshalJSON(b)
	t.NoError(err)

	t.Equal(mh.Hint(), unmarshaled.(Hinter).Hint())
	t.Equal(mh.A, unmarshaled.(customMarshalHinted).A)
	t.Equal(mh.B, unmarshaled.(customMarshalHinted).B)
}

func TestHintsetMarshal(t *testing.T) {
	suite.Run(t, new(testHintsetMarshal))
}
