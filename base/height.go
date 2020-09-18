package base

import (
	"fmt"
	"strconv"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	NilHeight        = Height(-2)
	PreGenesisHeight = Height(-1)
)

// Height stands for height of Block
type Height int64

func NewHeightFromString(s string) (Height, error) {
	if i, err := strconv.ParseInt(s, 10, 64); err != nil {
		return NilHeight, err
	} else {
		return Height(i), nil
	}
}

func NewHeightFromBytes(b []byte) (Height, error) {
	if i, err := util.BytesToInt64(b); err != nil {
		return NilHeight, err
	} else {
		return Height(i), nil
	}
}

// IsValid checks Height.
func (ht Height) IsValid([]byte) error {
	if ht < PreGenesisHeight {
		return isvalid.InvalidError.Errorf("height must be greater than %d; height=%d", PreGenesisHeight, ht)
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

func (ht Height) IsEmpty() bool {
	return ht == NilHeight
}
