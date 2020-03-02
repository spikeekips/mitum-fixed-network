package isaac

import (
	"fmt"

	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/util"
)

// Height stands for height of Block
type Height int64

// IsValid checks Height.
func (ht Height) IsValid([]byte) error {
	if ht < 0 {
		return isvalid.InvalidError.Wrapf("height must be greater than 0; height=%d", ht)
	}

	return nil
}

// Int64 returns int64 of height.
func (ht Height) Int64() int64 {
	return int64(ht)
}

func (ht Height) Bytes() []byte {
	return util.Int64ToBytes(int64(ht))
}

func (ht Height) String() string {
	return fmt.Sprintf("%d", ht)
}
