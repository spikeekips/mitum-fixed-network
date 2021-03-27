package storage

import (
	"os"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/errors"
)

var (
	TimeoutError = errors.NewError("timeout")
	StorageError = errors.NewError("storage error")
	FSError      = errors.NewError("fs error")
)

func WrapStorageError(err error) error {
	switch {
	case err == nil:
		return nil
	case xerrors.Is(err, util.FoundError):
	case xerrors.Is(err, util.NotFoundError):
	case xerrors.Is(err, util.DuplicatedError):
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
		return util.FoundError.Wrap(err)
	case os.IsNotExist(err):
		return util.NotFoundError.Wrap(err)
	case xerrors.Is(err, util.FoundError):
	case xerrors.Is(err, util.NotFoundError):
	case xerrors.Is(err, FSError):
	default:
		return FSError.Wrap(err)
	}

	return err
}
