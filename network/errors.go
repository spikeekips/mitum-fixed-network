package network

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/errors"
)

var NetworkError = errors.NewError("network error")

func WrapError(err error) error {
	if err == nil {
		return nil
	}

	if xerrors.Is(err, NetworkError) {
		return err
	}

	return NetworkError.Wrap(err)
}
