package util

import (
	"bufio"
	"context"
	"io"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
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
	sem := semaphore.NewWeighted(limit)
	eg, ctx := errgroup.WithContext(context.Background())

	for {
		b, err := get()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return err
		}

		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		eg.Go(func() error {
			defer sem.Release(1)

			if _, err := w.Write(append(b, []byte("\n")...)); err != nil {
				return err
			}

			return nil
		})
	}

	if err := sem.Acquire(ctx, limit); err != nil {
		if !errors.Is(err, context.Canceled) {
			return err
		}
	}

	return eg.Wait()
}
