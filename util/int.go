package util

import (
	"bytes"
	"encoding/binary"
	"math"
)

func IntToBytes(i int) []byte {
	b := new(bytes.Buffer)
	_ = binary.Write(b, binary.LittleEndian, int64(i))

	return b.Bytes()
}

func Int64ToBytes(i int64) []byte {
	b := new(bytes.Buffer)
	_ = binary.Write(b, binary.LittleEndian, i)

	return b.Bytes()
}

func UintToBytes(i uint) []byte {
	b := new(bytes.Buffer)
	_ = binary.Write(b, binary.LittleEndian, uint64(i))

	return b.Bytes()
}

func Uint64ToBytes(i uint64) []byte {
	b := new(bytes.Buffer)
	_ = binary.Write(b, binary.LittleEndian, i)

	return b.Bytes()
}

func Float64ToBytes(i float64) []byte {
	bt := math.Float64bits(i)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, bt)

	return bytes
}

func BytesToFloat64(b []byte) float64 {
	bt := binary.LittleEndian.Uint64(b)

	return math.Float64frombits(bt)
}
