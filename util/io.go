package util

import (
	"bufio"
	"context"
	"io"

	"github.com/pkg/errors"
)

type NilReadCloser struct {
	io.Reader
}

func NewNilReadCloser(r io.Reader) NilReadCloser {
	return NilReadCloser{Reader: r}
}

func (NilReadCloser) Close() error {
	return nil
}

func Readlines(r io.Reader, callback func([]byte) error) error {
	br := bufio.NewReader(r)
	for {
		l, err := br.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return err
		}

		if err := callback(l); err != nil {
			return err
		}
	}

	return nil
}

func Writeline(w io.Writer, get func() ([]byte, error)) error {
	for {
		if i, err := get(); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return err
		} else if _, err := w.Write(append(i, []byte("\n")...)); err != nil {
			return err
		}
	}

	return nil
}

func WritelineAsync(w io.Writer, get func() ([]byte, error), limit int64) error {
	wk := NewErrgroupWorker(context.Background(), limit)
	defer wk.Close()

	errch := make(chan error, 1)
	go func() {
		defer wk.Done()

		for {
			b, err := get()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}

				errch <- err

				return
			}

			if err := wk.NewJob(func(context.Context, uint64) error {
				_, err := w.Write(append(b, []byte("\n")...))

				return err
			}); err != nil {
				break
			}
		}

		errch <- nil
	}()

	err := wk.Wait()
	if cerr := <-errch; cerr != nil {
		return cerr
	}

	return err
}
