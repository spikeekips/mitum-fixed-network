package encoder

import (
	"github.com/spikeekips/mitum/hint"
)

// s0 is
// - not Hinter
// - don't have EncodeJSON
// - don't have DecodeJSON
type s0 struct {
	A string
}

// sH0 is
// - Hinter
// - has EncodeJSON
// - has DecodeJSON
type sH0 struct {
	A string
}

func (h0 sH0) Hint() hint.Hint {
	h, err := hint.NewHint(
		hint.Type([2]byte{0xff, 0x20}),
		"1.2.3",
	)
	if err != nil {
		panic(err)
	}

	return h
}

// sH1 is
// - Hinter
// - has EncodeJSON
// - don't have DecodeJSON
type sH1 struct {
	A string
	b int
}

func (h0 sH1) Hint() hint.Hint {
	h, err := hint.NewHint(
		hint.Type([2]byte{0xff, 0x20}),
		"0.1",
	)
	if err != nil {
		panic(err)
	}

	return h
}

// sH2 is
// - Hinter
// - has EncodeJSON; returns json.RawMessage
// - don't have DecodeJSON
type sH2 struct {
	A string
	B int
}

func (h0 sH2) Hint() hint.Hint {
	h, err := hint.NewHint(
		hint.Type([2]byte{0xff, 0x20}),
		"0.1",
	)
	if err != nil {
		panic(err)
	}

	return h
}

// sH3 is
// - Hinter
// - has EncodeJSON; returns []byte
// - don't have DecodeJSON
type sH3 struct {
	A string
	B int
}

func (h0 sH3) Hint() hint.Hint {
	h, err := hint.NewHint(
		hint.Type([2]byte{0xff, 0x20}),
		"0.1",
	)
	if err != nil {
		panic(err)
	}

	return h
}

// sH4 is
// - Hinter
// - has EncodeJSON; returns map[string]interface{}
// - don't have DecodeJSON
type sH4 struct {
	A string
	B int
}

func (h0 sH4) Hint() hint.Hint {
	h, err := hint.NewHint(
		hint.Type([2]byte{0xff, 0x20}),
		"0.1",
	)
	if err != nil {
		panic(err)
	}

	return h
}
