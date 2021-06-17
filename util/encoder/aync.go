package encoder

import "io"

type AsyncWriter struct {
	enc Encoder
	w   io.Writer
}

func NewAsyncWriter(enc Encoder, w io.Writer) *AsyncWriter {
	return &AsyncWriter{enc: enc, w: w}
}
