package util

import (
	"time"

	"github.com/spikeekips/mitum/util/errors"
	"golang.org/x/xerrors"
)

type Callbacker interface {
	Callback() error
}

var StopRetryingError = errors.NewError("stop retrying")

func Retry(max uint, interval time.Duration, callback func(int) error) error {
	var err error
	var tried int
	for {
		if max > 0 && uint(tried) == max { // if max == 0,  do forever
			break
		}

		if err = callback(tried); err == nil {
			return nil
		} else if xerrors.Is(err, StopRetryingError) {
			return err
		}

		tried++

		if interval > 0 {
			<-time.After(interval)
		}
	}

	return err
}
