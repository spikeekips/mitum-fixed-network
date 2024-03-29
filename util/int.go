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

func BytesToInt(b []byte) (int, error) {
	i, err := BytesToInt64(b)
	if err != nil {
		return 0, err
	}

	return int(i), nil
}

func Int64ToBytes(i int64) []byte {
	b := new(bytes.Buffer)
	_ = binary.Write(b, binary.LittleEndian, i)

	return b.Bytes()
}

func BytesToInt64(b []byte) (int64, error) {
	var i int64
	buf := bytes.NewReader(b)
	err := binary.Read(buf, binary.LittleEndian, &i)
	if err != nil {
		return 0, err
	}

	return i, nil
}

func UintToBytes(i uint) []byte {
	b := new(bytes.Buffer)
	_ = binary.Write(b, binary.LittleEndian, uint64(i))

	return b.Bytes()
}

func BytesToUint8(b []byte) (uint8, error) {
	var i uint8
	buf := bytes.NewReader(b)
	err := binary.Read(buf, binary.LittleEndian, &i)
	if err != nil {
		return 0, err
	}

	return i, nil
}

func Uint8ToBytes(i uint8) []byte {
	b := new(bytes.Buffer)
	_ = binary.Write(b, binary.LittleEndian, i)

	return b.Bytes()
}

func Uint64ToBytes(i uint64) []byte {
	b := new(bytes.Buffer)
	_ = binary.Write(b, binary.LittleEndian, i)

	return b.Bytes()
}

func BytesToUint64(b []byte) (uint64, error) {
	var i uint64
	buf := bytes.NewReader(b)
	err := binary.Read(buf, binary.LittleEndian, &i)
	if err != nil {
		return 0, err
	}

	return i, nil
}

func Float64ToBytes(i float64) []byte {
	bt := math.Float64bits(i)
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, bt)

	return b
}

func BytesToFloat64(b []byte) float64 {
	bt := binary.LittleEndian.Uint64(b)

	return math.Float64frombits(bt)
}
