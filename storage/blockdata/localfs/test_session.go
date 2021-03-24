// +build test

package localfs

import "github.com/spikeekips/mitum/base/block"

func (ss *Session) MapData() block.BaseBlockDataMap {
	return ss.mapData
}

func (ss *Session) Done() (block.BaseBlockDataMap, error) {
	return ss.done()
}
