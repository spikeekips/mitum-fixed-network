package state

import (
	"sync"

	"github.com/spikeekips/mitum/base"
)

var stateUpdaterPool = sync.Pool{
	New: func() interface{} {
		return new(StateUpdater)
	},
}

var (
	StateUpdaterPoolGet = func() *StateUpdater {
		return stateUpdaterPool.Get().(*StateUpdater)
	}
	StateUpdaterPoolPut = func(stu *StateUpdater) {
		stu.Lock()
		defer stu.Unlock()

		stu.State = nil
		stu.opcache = nil
		stu.orig = nil
		stu.height = base.NilHeight
		stu.operations = nil

		stateUpdaterPool.Put(stu)
	}
)
