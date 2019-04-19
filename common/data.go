package common

import "encoding"

type BinaryEncoder interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type TextEncoder interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}
