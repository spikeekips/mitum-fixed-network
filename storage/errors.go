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
	case xerrors.Is(err, FoundError):
	case xerrors.Is(err, NotFoundError):
	case xerrors.Is(err, DuplicatedError):
	case xerrors.Is(err, TimeoutError):
	case xerrors.Is(err, StorageError):
	default:
		return StorageError.Wrap(err)
	}

	return err
}

func WrapFSError(err error) error {
	switch {
	case err == nil:
		return nil
	case os.IsExist(err):
		return FoundError.Wrap(err)
	case os.IsNotExist(err):
		return NotFoundError.Wrap(err)
	case xerrors.Is(err, FoundError):
	case xerrors.Is(err, NotFoundError):
	case xerrors.Is(err, FSError):
	default:
		return FSError.Wrap(err)
	}

	return err
}
