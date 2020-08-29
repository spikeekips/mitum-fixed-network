package storage

import (
	"os"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/errors"
)

var (
	NotFoundError   = errors.NewError("not found")
	FoundError      = errors.NewError("but found")
	DuplicatedError = errors.NewError("duplicated error")
	TimeoutError    = errors.NewError("timeout")
	StorageError    = errors.NewError("storage error")
	FSError         = errors.NewError("fs error")
)

func WrapStorageError(err error) error {
	switch {
	case err == nil:
		return nil
	case IsNotFoundError(err):
		return err
	case IsDuplicatedError(err):
		return err
	case IsTimeoutError(err):
		return err
	case xerrors.Is(err, StorageError):
		return err
	default:
		return StorageError.Wrap(err)
	}
}

func IsFoundError(err error) bool {
	return xerrors.Is(err, FoundError)
}

func IsNotFoundError(err error) bool {
	return xerrors.Is(err, NotFoundError)
}

func IsDuplicatedError(err error) bool {
	return xerrors.Is(err, DuplicatedError)
}

func IsTimeoutError(err error) bool {
	return xerrors.Is(err, TimeoutError)
}

func WrapFSError(err error) error {
	switch {
	case err == nil:
		return nil
	case os.IsExist(err):
		return FoundError.Wrap(err)
	case os.IsNotExist(err):
		return NotFoundError.Wrap(err)
	case IsFoundError(err):
		return err
	case IsNotFoundError(err):
		return err
	case xerrors.Is(err, FSError):
		return err
	default:
		return FSError.Wrap(err)
	}
}
