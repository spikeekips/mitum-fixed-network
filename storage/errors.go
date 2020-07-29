package storage

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/errors"
)

var (
	NotFoundError   = errors.NewError("not found")
	DuplicatedError = errors.NewError("duplicated error")
	StorageError    = errors.NewError("storage error")
)

func WrapError(err error) error {
	switch {
	case err == nil:
		return nil
	case IsNotFoundError(err):
		return err
	case IsDuplicatedError(err):
		return err
	case xerrors.Is(err, StorageError):
		return err
	default:
		return StorageError.Wrap(err)
	}
}

func IsNotFoundError(err error) bool {
	return xerrors.Is(err, NotFoundError)
}

func IsDuplicatedError(err error) bool {
	return xerrors.Is(err, DuplicatedError)
}
