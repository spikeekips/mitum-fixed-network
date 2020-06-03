package logging

import (
	"net"
	"time"

	"github.com/rs/zerolog"
)

type Emitter interface {
	AnErr(string, error) Emitter
	Array(string, zerolog.LogArrayMarshaler) Emitter
	Bool(string, bool) Emitter
	Bools(string, []bool) Emitter
	Bytes(string, []byte) Emitter
	Dict(string, Emitter) Emitter
	Dur(string, time.Duration) Emitter
	Durs(string, []time.Duration) Emitter
	EmbedObject(zerolog.LogObjectMarshaler) Emitter
	Err(err error) Emitter
	Errs(string, []error) Emitter
	Fields(map[string]interface{}) Emitter
	Float32(string, float32) Emitter
	Float64(string, float64) Emitter
	Floats32(string, []float32) Emitter
	Floats64(string, []float64) Emitter
	Hex(string, []byte) Emitter
	IPAddr(string, net.IP) Emitter
	IPPrefix(string, net.IPNet) Emitter
	Int(string, int) Emitter
	Int16(string, int16) Emitter
	Int32(string, int32) Emitter
	Int64(string, int64) Emitter
	Int8(string, int8) Emitter
	Interface(string, interface{}) Emitter
	Ints(string, []int) Emitter
	Ints16(string, []int16) Emitter
	Ints32(string, []int32) Emitter
	Ints64(string, []int64) Emitter
	Ints8(string, []int8) Emitter
	MACAddr(string, net.HardwareAddr) Emitter
	Object(string, zerolog.LogObjectMarshaler) Emitter
	RawJSON(string, []byte) Emitter
	Str(string, string) Emitter
	Strs(string, []string) Emitter
	Time(string, time.Time) Emitter
	TimeDiff(string, time.Time, time.Time) Emitter
	Times(string, []time.Time) Emitter
	Uint(string, uint) Emitter
	Uint16(string, uint16) Emitter
	Uint32(string, uint32) Emitter
	Uint64(string, uint64) Emitter
	Uint8(string, uint8) Emitter
	Uints(string, []uint) Emitter
	Uints16(string, []uint16) Emitter
	Uints32(string, []uint32) Emitter
	Uints64(string, []uint64) Emitter
	Uints8(string, []uint8) Emitter
	Msg(string)
	Msgf(string, ...interface{})
	Send()
	Hinted(string, LogHintedMarshaler) Emitter
	HintedVerbose(string, LogHintedMarshaler, bool) Emitter
}
