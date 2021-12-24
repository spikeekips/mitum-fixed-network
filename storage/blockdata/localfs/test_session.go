// +build test

package localfs

import "github.com/spikeekips/mitum/base/block"

func (ss *Session) MapData() block.BaseBlockdataMap {
	return ss.mapData
}

func (ss *Session) Done() (block.BaseBlockdataMap, error) {
	return ss.done()
}
