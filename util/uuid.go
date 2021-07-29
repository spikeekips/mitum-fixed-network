package util

import (
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/oklog/ulid"
	uuid "github.com/satori/go.uuid"
)

var (
	ulidLock    = &sync.Mutex{}
	ulidEntropy io.Reader
	ulidT       = time.Unix(1000000, 0)
)

func init() {
	ulidEntropy = ulid.Monotonic(rand.New(rand.NewSource(ulidT.UnixNano())), 0) // nolint
}

func UUID() uuid.UUID {
	return uuid.NewV4()
}

func ULID() ulid.ULID {
	ulidLock.Lock()
	defer ulidLock.Unlock()

	return ulid.MustNew(ulid.Timestamp(ulidT), ulidEntropy)
}

func ULIDBytes() []byte {
	u := ULID()
	return u[:]
}
