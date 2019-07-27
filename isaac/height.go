package isaac

import (
	"github.com/spikeekips/mitum/common"
)

type Height struct {
	common.Big
}

func NewBlockHeight(height uint64) Height {
	return Height{Big: common.NewBigFromUint64(height)}
}

func (ht Height) Equal(height Height) bool {
	return ht.Big.Equal(height.Big)
}

func (ht Height) Add(v interface{}) Height {
	return Height{Big: ht.Big.Add(v)}
}

func (ht Height) SubOk(v interface{}) (Height, bool) {
	s := ht.Big.Sub(v)
	if s.Cmp(0) < 0 {
		return Height{}, false
	}

	return Height{Big: s}, true
}

func (ht Height) Cmp(height Height) int {
	return ht.Big.Cmp(height.Big)
}
