package util

import (
	"time"

	"github.com/pkg/errors"
)

type Callbacker interface {
	Callback() error
}

var StopRetryingError = NewError("stop retrying")

func Retry(max uint, interval time.Duration, callback func(int) error) error {
	var err error
	var tried int
	for {
		if max > 0 && uint(tried) == max { // if max == 0,  do forever
			break
		}

		if err = callback(tried); err == nil {
			return nil
		} else if errors.Is(err, StopRetryingError) {
			return err
		}

		tried++

		if interval > 0 {
			<-time.After(interval)
		}
	}

	return err
}
