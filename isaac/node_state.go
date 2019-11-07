package isaac

import (
	"encoding/json"
	"sync"

	"github.com/rs/zerolog"

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

func (hs *HomeState) MarshalJSON() ([]byte, error) {
	hs.RLock()
	defer hs.RUnlock()

	return json.Marshal(map[string]interface{}{
		"home":           hs.home,
		"block":          hs.block,
		"previous_block": hs.previousBlock,
		"state":          hs.state.String(),
	})
}

func (hs *HomeState) MarshalZerologObject(e *zerolog.Event) {
	hs.RLock()
	defer hs.RUnlock()

	e.Object("home", hs.home)
	e.Object("block", hs.block)
	e.Object("previous_block", hs.previousBlock)
	e.Str("state", hs.state.String())
}
