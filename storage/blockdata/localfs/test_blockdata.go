// +build test

package localfs

import (
	"github.com/spikeekips/mitum/base"
)

func (st *Blockdata) CreateDirectory(p string) error {
	return st.createDirectory(p)
}

func (st *Blockdata) HeightDirectory(height base.Height, abs bool) string {
	return st.heightDirectory(height, abs)
}
