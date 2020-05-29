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

func newEmitterFromEvent(ze *zerolog.Event) Emitter {
	return newEvent(ze)
}

func (e *Event) event(ze *zerolog.Event) Emitter {
	return newEmitterFromEvent(ze)
}

func (e *Event) AnErr(key string, err error) Emitter {
	return e.event(e.Event.AnErr(key, err))
}

func (e *Event) Array(key string, arr zerolog.LogArrayMarshaler) Emitter {
	return e.event(e.Event.Array(key, arr))
}

func (e *Event) Bool(key string, b bool) Emitter {
	return e.event(e.Event.Bool(key, b))
}

func (e *Event) Bools(key string, b []bool) Emitter {
	return e.event(e.Event.Bools(key, b))
}

func (e *Event) Bytes(key string, val []byte) Emitter {
	return e.event(e.Event.Bytes(key, val))
}

func (e *Event) Caller(skip ...int) *Event {
	return e.event(e.Event.Caller(skip...)).(*Event)
}

func (e *Event) Dict(key string, dict Emitter) Emitter {
	return e.event(e.Event.Dict(key, dict.(*Event).Event))
}

func (e *Event) Dur(key string, d time.Duration) Emitter {
	return e.event(e.Event.Dur(key, d))
}

func (e *Event) Durs(key string, d []time.Duration) Emitter {
	return e.event(e.Event.Durs(key, d))
}

func (e *Event) EmbedObject(obj zerolog.LogObjectMarshaler) Emitter {
	return e.event(e.Event.EmbedObject(obj))
}

func (e *Event) Err(err error) Emitter {
	return e.event(e.Event.Err(err))
}

func (e *Event) Errs(key string, errs []error) Emitter {
	return e.event(e.Event.Errs(key, errs))
}

func (e *Event) Fields(fields map[string]interface{}) Emitter {
	return e.event(e.Event.Fields(fields))
}

func (e *Event) Float32(key string, f float32) Emitter {
	return e.event(e.Event.Float32(key, f))
}

func (e *Event) Float64(key string, f float64) Emitter {
	return e.event(e.Event.Float64(key, f))
}

func (e *Event) Floats32(key string, f []float32) Emitter {
	return e.event(e.Event.Floats32(key, f))
}

func (e *Event) Floats64(key string, f []float64) Emitter {
	return e.event(e.Event.Floats64(key, f))
}

func (e *Event) Hex(key string, val []byte) Emitter {
	return e.event(e.Event.Hex(key, val))
}

func (e *Event) IPAddr(key string, ip net.IP) Emitter {
	return e.event(e.Event.IPAddr(key, ip))
}

func (e *Event) IPPrefix(key string, pfx net.IPNet) Emitter {
	return e.event(e.Event.IPPrefix(key, pfx))
}

func (e *Event) Int(key string, i int) Emitter {
	return e.event(e.Event.Int(key, i))
}

func (e *Event) Int16(key string, i int16) Emitter {
	return e.event(e.Event.Int16(key, i))
}

func (e *Event) Int32(key string, i int32) Emitter {
	return e.event(e.Event.Int32(key, i))
}

func (e *Event) Int64(key string, i int64) Emitter {
	return e.event(e.Event.Int64(key, i))
}

func (e *Event) Int8(key string, i int8) Emitter {
	return e.event(e.Event.Int8(key, i))
}

func (e *Event) Interface(key string, i interface{}) Emitter {
	return e.event(e.Event.Interface(key, i))
}

func (e *Event) Ints(key string, i []int) Emitter {
	return e.event(e.Event.Ints(key, i))
}

func (e *Event) Ints16(key string, i []int16) Emitter {
	return e.event(e.Event.Ints16(key, i))
}

func (e *Event) Ints32(key string, i []int32) Emitter {
	return e.event(e.Event.Ints32(key, i))
}

func (e *Event) Ints64(key string, i []int64) Emitter {
	return e.event(e.Event.Ints64(key, i))
}

func (e *Event) Ints8(key string, i []int8) Emitter {
	return e.event(e.Event.Ints8(key, i))
}

func (e *Event) MACAddr(key string, ha net.HardwareAddr) Emitter {
	return e.event(e.Event.MACAddr(key, ha))
}

func (e *Event) Object(key string, obj zerolog.LogObjectMarshaler) Emitter {
	return e.event(e.Event.Object(key, obj))
}

func (e *Event) RawJSON(key string, b []byte) Emitter {
	return e.event(e.Event.RawJSON(key, b))
}

func (e *Event) Str(key, val string) Emitter {
	return e.event(e.Event.Str(key, val))
}

func (e *Event) Strs(key string, vals []string) Emitter {
	return e.event(e.Event.Strs(key, vals))
}

func (e *Event) Time(key string, t time.Time) Emitter {
	return e.event(e.Event.Time(key, t))
}

func (e *Event) TimeDiff(key string, t, start time.Time) Emitter {
	return e.event(e.Event.TimeDiff(key, t, start))
}

func (e *Event) Times(key string, t []time.Time) Emitter {
	return e.event(e.Event.Times(key, t))
}

func (e *Event) Uint(key string, i uint) Emitter {
	return e.event(e.Event.Uint(key, i))
}

func (e *Event) Uint16(key string, i uint16) Emitter {
	return e.event(e.Event.Uint16(key, i))
}

func (e *Event) Uint32(key string, i uint32) Emitter {
	return e.event(e.Event.Uint32(key, i))
}

func (e *Event) Uint64(key string, i uint64) Emitter {
	return e.event(e.Event.Uint64(key, i))
}

func (e *Event) Uint8(key string, i uint8) Emitter {
	return e.event(e.Event.Uint8(key, i))
}

func (e *Event) Uints(key string, i []uint) Emitter {
	return e.event(e.Event.Uints(key, i))
}

func (e *Event) Uints16(key string, i []uint16) Emitter {
	return e.event(e.Event.Uints16(key, i))
}

func (e *Event) Uints32(key string, i []uint32) Emitter {
	return e.event(e.Event.Uints32(key, i))
}

func (e *Event) Uints64(key string, i []uint64) Emitter {
	return e.event(e.Event.Uints64(key, i))
}

func (e *Event) Uints8(key string, i []uint8) Emitter {
	return e.event(e.Event.Uints8(key, i))
}

func (e *Event) Hinted(key string, obj LogHintedMarshaler) Emitter {
	if obj == nil {
		return e.Str(key, "")
	}

	_ = obj.MarshalLog(key, e, false)

	return e
}

func (e *Event) HintedVerbose(key string, obj LogHintedMarshaler, verbose bool) Emitter {
	if obj == nil {
		return e.Str(key, "")
	}

	_ = obj.MarshalLog(key, e, verbose)

	return e
}
