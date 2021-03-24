// +build test

package localfs

import (
	"github.com/spikeekips/mitum/base"
)

func (st *BlockData) CreateDirectory(p string) error {
	return st.createDirectory(p)
}

func (st *BlockData) HeightDirectory(height base.Height, abs bool) string {
	return st.heightDirectory(height, abs)
}
