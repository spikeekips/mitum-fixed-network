package logging

import (
	"net"
	"time"

	"github.com/rs/zerolog"
)

// Event wraps the original zerolog.Event to support value-based marshaling.
type Event struct {
	*zerolog.Event
}

func Dict() *Event {
	return newEvent(zerolog.Dict())
}

func newEvent(ze *zerolog.Event) *Event {
	return &Event{Event: ze}
}

func (e *Event) AnErr(key string, err error) Emitter {
	e.Event.AnErr(key, err)
	return e
}

func (e *Event) Array(key string, arr zerolog.LogArrayMarshaler) Emitter {
	e.Event.Array(key, arr)
	return e
}

func (e *Event) Bool(key string, b bool) Emitter {
	e.Event.Bool(key, b)
	return e
}

func (e *Event) Bools(key string, b []bool) Emitter {
	e.Event.Bools(key, b)
	return e
}

func (e *Event) Bytes(key string, val []byte) Emitter {
	e.Event.Bytes(key, val)
	return e
}

func (e *Event) CallerWithSkipFrameCount(skip int) Emitter {
	e.Event.Caller(skip)

	return e
}

func (e *Event) Dict(key string, dict Emitter) Emitter {
	e.Event.Dict(key, dict.(*Event).Event)
	return e
}

func (e *Event) Dur(key string, d time.Duration) Emitter {
	e.Event.Dur(key, d)
	return e
}

func (e *Event) Durs(key string, d []time.Duration) Emitter {
	e.Event.Durs(key, d)
	return e
}

func (e *Event) EmbedObject(obj zerolog.LogObjectMarshaler) Emitter {
	e.Event.EmbedObject(obj)
	return e
}

func (e *Event) Err(err error) Emitter {
	e.Event.Err(err)
	return e
}

func (e *Event) Errs(key string, errs []error) Emitter {
	e.Event.Errs(key, errs)
	return e
}

func (e *Event) Fields(fields map[string]interface{}) Emitter {
	e.Event.Fields(fields)
	return e
}

func (e *Event) Float32(key string, f float32) Emitter {
	e.Event.Float32(key, f)
	return e
}

func (e *Event) Float64(key string, f float64) Emitter {
	e.Event.Float64(key, f)
	return e
}

func (e *Event) Floats32(key string, f []float32) Emitter {
	e.Event.Floats32(key, f)
	return e
}

func (e *Event) Floats64(key string, f []float64) Emitter {
	e.Event.Floats64(key, f)
	return e
}

func (e *Event) Hex(key string, val []byte) Emitter {
	e.Event.Hex(key, val)
	return e
}

func (e *Event) IPAddr(key string, ip net.IP) Emitter {
	e.Event.IPAddr(key, ip)
	return e
}

func (e *Event) IPPrefix(key string, pfx net.IPNet) Emitter {
	e.Event.IPPrefix(key, pfx)
	return e
}

func (e *Event) Int(key string, i int) Emitter {
	e.Event.Int(key, i)
	return e
}

func (e *Event) Int16(key string, i int16) Emitter {
	e.Event.Int16(key, i)
	return e
}

func (e *Event) Int32(key string, i int32) Emitter {
	e.Event.Int32(key, i)
	return e
}

func (e *Event) Int64(key string, i int64) Emitter {
	e.Event.Int64(key, i)
	return e
}

func (e *Event) Int8(key string, i int8) Emitter {
	e.Event.Int8(key, i)
	return e
}

func (e *Event) Interface(key string, i interface{}) Emitter {
	e.Event.Interface(key, i)
	return e
}

func (e *Event) Ints(key string, i []int) Emitter {
	e.Event.Ints(key, i)
	return e
}

func (e *Event) Ints16(key string, i []int16) Emitter {
	e.Event.Ints16(key, i)
	return e
}

func (e *Event) Ints32(key string, i []int32) Emitter {
	e.Event.Ints32(key, i)
	return e
}

func (e *Event) Ints64(key string, i []int64) Emitter {
	e.Event.Ints64(key, i)
	return e
}

func (e *Event) Ints8(key string, i []int8) Emitter {
	e.Event.Ints8(key, i)
	return e
}

func (e *Event) MACAddr(key string, ha net.HardwareAddr) Emitter {
	e.Event.MACAddr(key, ha)
	return e
}

func (e *Event) Object(key string, obj zerolog.LogObjectMarshaler) Emitter {
	e.Event.Object(key, obj)
	return e
}

func (e *Event) RawJSON(key string, b []byte) Emitter {
	e.Event.RawJSON(key, b)
	return e
}

func (e *Event) Str(key, val string) Emitter {
	e.Event.Str(key, val)
	return e
}

func (e *Event) Strs(key string, vals []string) Emitter {
	e.Event.Strs(key, vals)
	return e
}

func (e *Event) TimeDiff(key string, t, start time.Time) Emitter {
	e.Event.TimeDiff(key, t, start)
	return e
}

func (e *Event) Time(key string, t time.Time) Emitter {
	e.Event.Time(key, t)
	return e
}

func (e *Event) Times(key string, t []time.Time) Emitter {
	e.Event.Times(key, t)
	return e
}

func (e *Event) Uint(key string, i uint) Emitter {
	e.Event.Uint(key, i)
	return e
}

func (e *Event) Uint16(key string, i uint16) Emitter {
	e.Event.Uint16(key, i)
	return e
}

func (e *Event) Uint32(key string, i uint32) Emitter {
	e.Event.Uint32(key, i)
	return e
}

func (e *Event) Uint64(key string, i uint64) Emitter {
	e.Event.Uint64(key, i)
	return e
}

func (e *Event) Uint8(key string, i uint8) Emitter {
	e.Event.Uint8(key, i)
	return e
}

func (e *Event) Uints(key string, i []uint) Emitter {
	e.Event.Uints(key, i)
	return e
}

func (e *Event) Uints16(key string, i []uint16) Emitter {
	e.Event.Uints16(key, i)
	return e
}

func (e *Event) Uints32(key string, i []uint32) Emitter {
	e.Event.Uints32(key, i)
	return e
}

func (e *Event) Uints64(key string, i []uint64) Emitter {
	e.Event.Uints64(key, i)
	return e
}

func (e *Event) Uints8(key string, i []uint8) Emitter {
	e.Event.Uints8(key, i)
	return e
}

func (e *Event) Hinted(key string, obj LogHintedMarshaler) Emitter {
	if obj == nil {
		return e.Interface(key, nil)
	}

	_ = obj.MarshalLog(key, e, false)

	return e
}

func (e *Event) HintedVerbose(key string, obj LogHintedMarshaler, verbose bool) Emitter {
	if obj == nil {
		return e.Interface(key, nil)
	}

	_ = obj.MarshalLog(key, e, verbose)

	return e
}
