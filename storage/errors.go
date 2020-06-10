package storage

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/errors"
)

var (
	NotFoundError = errors.NewError("not found")
	StorageError  = errors.NewError("storage error")
)

func WrapError(err error) error {
	if err == nil {
		return nil
	}

	if IsNotFoundError(err) {
		return err
	} else if xerrors.Is(err, StorageError) {
		return err
	}

	return StorageError.Wrap(err)
}

func IsNotFoundError(err error) bool {
	return xerrors.Is(err, NotFoundError)
}
