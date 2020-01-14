package encoder

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/spikeekips/mitum/hint"
)

type bH struct {
	A string
	B int
	C interface{}
}

func (bh bH) Hint() hint.Hint {
	h, err := hint.NewHint(
		hint.Type([2]byte{0xff, 0x20}),
		"1.2.3",
	)
	if err != nil {
		panic(err)
	}

	return h
}

func (bh bH) EncodeJSON(enc *HintEncoder) (interface{}, error) {
	var o interface{}
	if bh.C != nil {
		b, err := bh.C.(*bH).EncodeJSON(enc)
		if err != nil {
			return nil, err
		}
		o = b
	}

	return &struct {
		JSONHinterHead
		A string
		B int
		C interface{}
	}{
		JSONHinterHead: NewJSONHinterHead(bh.Hint()),
		A:              bh.A,
		B:              bh.B,
		C:              o,
	}, nil
}

type bJ struct {
	A string
	B int
	C interface{}
}

func (bj bJ) Hint() hint.Hint {
	h, err := hint.NewHint(
		hint.Type([2]byte{0xff, 0x21}),
		"1.2.3",
	)
	if err != nil {
		panic(err)
	}

	return h
}

func (bj bJ) MarshalJSON() ([]byte, error) {
	return (JSON{}).Marshal(struct {
		JSONHinterHead
		A string
		B int
		C interface{}
	}{
		JSONHinterHead: NewJSONHinterHead(bj.Hint()),
		A:              bj.A,
		B:              bj.B,
		C:              bj.C,
	})
}

func benchmarkNestedHinterEncode(n int, bench *testing.B) {
	je := NewHintEncoder(JSON{})
	encs := NewEncoders()
	_ = hint.RegisterType((bH{}).Hint().Type(), "hinted-struct-for-benchmark")
	_ = encs.Add(je)
	_ = encs.AddHinter(bH{})

	s := bH{
		A: uuid.Must(uuid.NewV4(), nil).String(),
		B: rand.Intn(100),
	}

	for i := 0; i < bench.N; i++ {
		started := time.Now()
		var last *bH = &s
		for j := 0; j < n; j++ {
			a := &bH{
				A: uuid.Must(uuid.NewV4(), nil).String(),
				B: rand.Intn(100),
			}
			last.C = a

			last = a
		}
		_, err := je.Encode(s)

		fmt.Fprintln(ioutil.Discard, ">", i, n, err == nil, time.Since(started))
	}
}

func benchmarkNestedJSONMarshal(n int, bench *testing.B) {
	je := JSON{}
	_ = hint.RegisterType((bJ{}).Hint().Type(), "bare-struct-for-benchmark")

	s := bJ{
		A: uuid.Must(uuid.NewV4(), nil).String(),
		B: rand.Intn(100),
	}

	for i := 0; i < bench.N; i++ {
		started := time.Now()
		var last *bJ = &s
		for j := 0; j < n; j++ {
			a := &bJ{
				A: uuid.Must(uuid.NewV4(), nil).String(),
				B: rand.Intn(100),
			}
			last.C = a

			last = a
		}
		_, err := je.Marshal(s)

		fmt.Fprintln(ioutil.Discard, ">", i, n, err == nil, time.Since(started))
	}
}

func BenchmarkNestedHinterEncode10(b *testing.B)   { benchmarkNestedHinterEncode(10, b) }
func BenchmarkNestedHinterEncode50(b *testing.B)   { benchmarkNestedHinterEncode(50, b) }
func BenchmarkNestedHinterEncode100(b *testing.B)  { benchmarkNestedHinterEncode(100, b) }
func BenchmarkNestedHinterEncode500(b *testing.B)  { benchmarkNestedHinterEncode(500, b) }
func BenchmarkNestedHinterEncode1000(b *testing.B) { benchmarkNestedHinterEncode(1000, b) }

func BenchmarkNestedJSONMarshal10(b *testing.B)   { benchmarkNestedJSONMarshal(10, b) }
func BenchmarkNestedJSONMarshal50(b *testing.B)   { benchmarkNestedJSONMarshal(50, b) }
func BenchmarkNestedJSONMarshal100(b *testing.B)  { benchmarkNestedJSONMarshal(100, b) }
func BenchmarkNestedJSONMarshal500(b *testing.B)  { benchmarkNestedJSONMarshal(500, b) }
func BenchmarkNestedJSONMarshal1000(b *testing.B) { benchmarkNestedJSONMarshal(1000, b) }
