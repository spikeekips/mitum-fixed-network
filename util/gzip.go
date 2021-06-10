package util

import (
	"compress/gzip"
	"io"
	"sync"
)

// GzipWriter closes the underlying reader too.
type GzipWriter struct {
	sync.Mutex
	*gzip.Writer
	f io.Writer
}

func NewGzipWriter(f io.Writer) *GzipWriter {
	return &GzipWriter{f: f, Writer: gzip.NewWriter(f)}
}

func (w *GzipWriter) Write(p []byte) (int, error) {
	w.Lock()
	defer w.Unlock()

	return w.Writer.Write(p)
}

func (w *GzipWriter) Close() error {
	if err := w.Writer.Close(); err != nil {
		return err
	}

	if j, ok := w.f.(io.Closer); !ok {
		return nil
	} else if err := j.Close(); err != nil {
		return err
	}

	return nil
}

// GzipReader closes the underlying reader too.
type GzipReader struct {
	*gzip.Reader
	f io.Reader
}

func NewGzipReader(f io.Reader) (GzipReader, error) {
	r, err := gzip.NewReader(f)
	if err != nil {
		return GzipReader{}, err
	}
	return GzipReader{f: f, Reader: r}, nil
}

func (r GzipReader) Close() error {
	if err := r.Reader.Close(); err != nil {
		return err
	}

	if j, ok := r.f.(io.Closer); !ok {
		return nil
	} else if err := j.Close(); err != nil {
		return err
	}

	return nil
}
