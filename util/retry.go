package util

import (
	"time"

	"github.com/spikeekips/mitum/util/errors"
	"golang.org/x/xerrors"
)

var StopRetryingError = errors.NewError("stop retrying")

func Retry(max uint, interval time.Duration, callback func() error) error {
	var err error
	var tried uint = 0
	for {
		if max > 0 && tried == max { // if max == 0,  do forever
			break
		}

		if err = callback(); err == nil {
			return nil
		} else if xerrors.Is(err, StopRetryingError) {
			return nil
		}

		if max > 0 {
			tried++
		}

		if interval > 0 {
			<-time.After(interval)
		}
	}

	return err
}
