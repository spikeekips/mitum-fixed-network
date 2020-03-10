package util

import (
	"io"
	"math/rand"
	"time"

	"github.com/oklog/ulid"
	uuid "github.com/satori/go.uuid"
)

var (
	ulidEntropy io.Reader
	ulidT       time.Time
)

func init() {
	ulidT = time.Unix(1000000, 0)
	ulidEntropy = ulid.Monotonic(rand.New(rand.NewSource(ulidT.UnixNano())), 0)
}

func UUID() uuid.UUID {
	return uuid.Must(uuid.NewV4(), nil)
}

func ULID() ulid.ULID {
	return ulid.MustNew(ulid.Timestamp(ulidT), ulidEntropy)
}

func ULIDBytes() []byte {
	u := ULID()
	return u[:]
}
