package util

import "time"

func Retry(max uint, interval time.Duration, callback func() error) error {
	var err error
	var tried uint = 0
	for {
		if max > 0 && tried == max { // if max == 0,  do forever
			break
		}

		if err = callback(); err == nil {
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
