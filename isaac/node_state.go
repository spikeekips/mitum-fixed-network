package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/node"
)

type HomeState struct {
	sync.RWMutex
	home          node.Home
	block         Block
	previousBlock Block
	state         node.State
}

func NewHomeState(home node.Home, block Block) *HomeState {
	return &HomeState{
		home:  home,
		block: block,
		state: node.StateBooting,
	}
}

func (hs *HomeState) Home() node.Home {
	return hs.home
}

func (hs *HomeState) Block() Block {
	hs.RLock()
	defer hs.RUnlock()

	return hs.block
}

func (hs *HomeState) PreviousBlock() Block {
	hs.RLock()
	defer hs.RUnlock()

	return hs.previousBlock
}

func (hs *HomeState) SetBlock(block Block) *HomeState {
	hs.Lock()
	defer hs.Unlock()

	hs.previousBlock = hs.block
	hs.block = block

	return hs
}

func (hs *HomeState) State() node.State {
	hs.RLock()
	defer hs.RUnlock()

	return hs.state
}

func (hs *HomeState) SetState(state node.State) *HomeState {
	hs.Lock()
	defer hs.Unlock()

	hs.state = state

	return hs
}