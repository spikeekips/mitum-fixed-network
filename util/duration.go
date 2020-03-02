package util

import (
	"time"
)

func DurationToBytes(d time.Duration) []byte {
	return Int64ToBytes(int64(d))
}
