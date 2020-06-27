package logging

import (
	"net"
	"time"

	"github.com/rs/zerolog"
)

type Context struct {
	zerolog.Context
	verbose bool
}

func newContext(c zerolog.Context, verbose bool) Context {
	return Context{Context: c, verbose: verbose}
}

func (c Context) AnErr(key string, err error) Emitter {
	return newContext(c.Context.AnErr(key, err), c.verbose)
}

func (c Context) Array(key string, arr zerolog.LogArrayMarshaler) Emitter {
	return newContext(c.Context.Array(key, arr), c.verbose)
}

func (c Context) Bool(key string, b bool) Emitter {
	return newContext(c.Context.Bool(key, b), c.verbose)
}

func (c Context) Bools(key string, b []bool) Emitter {
	return newContext(c.Context.Bools(key, b), c.verbose)
}

func (c Context) Bytes(key string, val []byte) Emitter {
	return newContext(c.Context.Bytes(key, val), c.verbose)
}

func (c Context) Caller() Context {
	return newContext(c.Context.Caller(), c.verbose)
}

func (c Context) CallerWithSkipFrameCount(skipFrameCount int) Emitter {
	return newContext(c.Context.CallerWithSkipFrameCount(skipFrameCount), c.verbose)
}

func (c Context) Dict(key string, dict Emitter) Emitter {
	return newContext(c.Context.Dict(key, dict.(*Event).Event), c.verbose)
}

func (c Context) Dur(key string, d time.Duration) Emitter {
	return newContext(c.Context.Dur(key, d), c.verbose)
}

func (c Context) Durs(key string, d []time.Duration) Emitter {
	return newContext(c.Context.Durs(key, d), c.verbose)
}

func (c Context) EmbedObject(obj zerolog.LogObjectMarshaler) Emitter {
	return newContext(c.Context.EmbedObject(obj), c.verbose)
}

func (c Context) Err(err error) Emitter {
	return newContext(c.Context.Err(err), c.verbose)
}

func (c Context) Errs(key string, errs []error) Emitter {
	return newContext(c.Context.Errs(key, errs), c.verbose)
}

func (c Context) Fields(fields map[string]interface{}) Emitter {
	return newContext(c.Context.Fields(fields), c.verbose)
}

func (c Context) Float32(key string, f float32) Emitter {
	return newContext(c.Context.Float32(key, f), c.verbose)
}

func (c Context) Float64(key string, f float64) Emitter {
	return newContext(c.Context.Float64(key, f), c.verbose)
}

func (c Context) Floats32(key string, f []float32) Emitter {
	return newContext(c.Context.Floats32(key, f), c.verbose)
}

func (c Context) Floats64(key string, f []float64) Emitter {
	return newContext(c.Context.Floats64(key, f), c.verbose)
}

func (c Context) Hex(key string, val []byte) Emitter {
	return newContext(c.Context.Hex(key, val), c.verbose)
}

func (c Context) IPAddr(key string, ip net.IP) Emitter {
	return newContext(c.Context.IPAddr(key, ip), c.verbose)
}

func (c Context) IPPrefix(key string, pfx net.IPNet) Emitter {
	return newContext(c.Context.IPPrefix(key, pfx), c.verbose)
}

func (c Context) Int(key string, i int) Emitter {
	return newContext(c.Context.Int(key, i), c.verbose)
}

func (c Context) Int16(key string, i int16) Emitter {
	return newContext(c.Context.Int16(key, i), c.verbose)
}

func (c Context) Int32(key string, i int32) Emitter {
	return newContext(c.Context.Int32(key, i), c.verbose)
}

func (c Context) Int64(key string, i int64) Emitter {
	return newContext(c.Context.Int64(key, i), c.verbose)
}

func (c Context) Int8(key string, i int8) Emitter {
	return newContext(c.Context.Int8(key, i), c.verbose)
}

func (c Context) Interface(key string, i interface{}) Emitter {
	return newContext(c.Context.Interface(key, i), c.verbose)
}

func (c Context) Ints(key string, i []int) Emitter {
	return newContext(c.Context.Ints(key, i), c.verbose)
}

func (c Context) Ints16(key string, i []int16) Emitter {
	return newContext(c.Context.Ints16(key, i), c.verbose)
}

func (c Context) Ints32(key string, i []int32) Emitter {
	return newContext(c.Context.Ints32(key, i), c.verbose)
}

func (c Context) Ints64(key string, i []int64) Emitter {
	return newContext(c.Context.Ints64(key, i), c.verbose)
}

func (c Context) Ints8(key string, i []int8) Emitter {
	return newContext(c.Context.Ints8(key, i), c.verbose)
}

func (c Context) MACAddr(key string, ha net.HardwareAddr) Emitter {
	return newContext(c.Context.MACAddr(key, ha), c.verbose)
}

func (c Context) Object(key string, obj zerolog.LogObjectMarshaler) Emitter {
	return newContext(c.Context.Object(key, obj), c.verbose)
}

func (c Context) RawJSON(key string, b []byte) Emitter {
	return newContext(c.Context.RawJSON(key, b), c.verbose)
}

func (c Context) Stack() Emitter {
	return newContext(c.Context.Stack(), c.verbose)
}

func (c Context) Str(key, val string) Emitter {
	return newContext(c.Context.Str(key, val), c.verbose)
}

func (c Context) Strs(key string, vals []string) Emitter {
	return newContext(c.Context.Strs(key, vals), c.verbose)
}

func (c Context) Time(key string, t time.Time) Emitter {
	return newContext(c.Context.Time(key, t), c.verbose)
}

func (c Context) TimeDiff(_ string, _, _ time.Time) Emitter {
	return c
}

func (c Context) Times(key string, t []time.Time) Emitter {
	return newContext(c.Context.Times(key, t), c.verbose)
}

func (c Context) Timestamp() Emitter {
	return newContext(c.Context.Timestamp(), c.verbose)
}

func (c Context) Uint(key string, i uint) Emitter {
	return newContext(c.Context.Uint(key, i), c.verbose)
}

func (c Context) Uint16(key string, i uint16) Emitter {
	return newContext(c.Context.Uint16(key, i), c.verbose)
}

func (c Context) Uint32(key string, i uint32) Emitter {
	return newContext(c.Context.Uint32(key, i), c.verbose)
}

func (c Context) Uint64(key string, i uint64) Emitter {
	return newContext(c.Context.Uint64(key, i), c.verbose)
}

func (c Context) Uint8(key string, i uint8) Emitter {
	return newContext(c.Context.Uint8(key, i), c.verbose)
}

func (c Context) Uints(key string, i []uint) Emitter {
	return newContext(c.Context.Uints(key, i), c.verbose)
}

func (c Context) Uints16(key string, i []uint16) Emitter {
	return newContext(c.Context.Uints16(key, i), c.verbose)
}

func (c Context) Uints32(key string, i []uint32) Emitter {
	return newContext(c.Context.Uints32(key, i), c.verbose)
}

func (c Context) Uints64(key string, i []uint64) Emitter {
	return newContext(c.Context.Uints64(key, i), c.verbose)
}

func (c Context) Uints8(key string, i []uint8) Emitter {
	return newContext(c.Context.Uints8(key, i), c.verbose)
}

func (c Context) Msg(string)                  {}
func (c Context) Msgf(string, ...interface{}) {}
func (c Context) Send()                       {}

func (c Context) Hinted(key string, obj LogHintedMarshaler) Emitter {
	if obj == nil {
		return c.Str(key, "")
	}

	return obj.MarshalLog(key, c, false)
}

func (c Context) HintedVerbose(key string, obj LogHintedMarshaler, verbose bool) Emitter {
	if obj == nil {
		return c.Str(key, "")
	}

	return obj.MarshalLog(key, c, verbose)
}
