package storage

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
)

var (
	TimeoutError = util.NewError("timeout")
	StorageError = util.NewError("storage error")
	FSError      = util.NewError("fs error")
)

func MergeStorageError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, util.FoundError):
	case errors.Is(err, util.NotFoundError):
	case errors.Is(err, util.DuplicatedError):
	case errors.Is(err, TimeoutError):
	case errors.Is(err, StorageError):
	default:
		return StorageError.Merge(err)
	}

	return err
}

func MergeFSError(err error) error {
	switch {
	case err == nil:
		return nil
	case os.IsExist(err):
		return util.FoundError.Merge(err)
	case os.IsNotExist(err):
		return util.NotFoundError.Merge(err)
	case errors.Is(err, util.FoundError):
	case errors.Is(err, util.NotFoundError):
	case errors.Is(err, FSError):
	default:
		return FSError.Merge(err)
	}

	return err
}
