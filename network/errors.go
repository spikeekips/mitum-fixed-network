package network

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
)

var (
	NetworkError          = util.NewError("network error")
	HandoverRejectedError = util.NewError("handover failed")
)

func MergeError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, NetworkError) {
		return err
	}

	return NetworkError.Merge(err)
}
