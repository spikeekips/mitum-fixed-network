package util

import (
	"io"
)

type NilReadCloser struct {
	io.Reader
}

func NewNilReadCloser(r io.Reader) NilReadCloser {
	return NilReadCloser{Reader: r}
}

func (rc NilReadCloser) Close() error {
	return nil
}
