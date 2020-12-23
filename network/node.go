package network

import (
	"sync"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
)

type Node interface {
	base.Node
	Channel() Channel
}

type LocalNode struct {
	sync.RWMutex
	base.BaseNodeV0
	privatekey key.Privatekey
	channel    Channel
}

func NewLocalNode(address base.Address, privatekey key.Privatekey, url string) *LocalNode {
	return &LocalNode{
		BaseNodeV0: base.NewBaseNodeV0(address, privatekey.Publickey(), url),
		privatekey: privatekey,
	}
}

func (ln *LocalNode) Publickey() key.Publickey {
	ln.RLock()
	defer ln.RUnlock()

	return ln.BaseNodeV0.Publickey()
}

func (ln *LocalNode) Privatekey() key.Privatekey {
	ln.RLock()
	defer ln.RUnlock()

	return ln.privatekey
}

func (ln *LocalNode) Channel() Channel {
	ln.RLock()
	defer ln.RUnlock()

	return ln.channel
}

func (ln *LocalNode) SetChannel(channel Channel) *LocalNode {
	ln.Lock()
	defer ln.Unlock()

	ln.channel = channel

	return ln
}

type RemoteNode struct {
	sync.RWMutex
	base.BaseNodeV0
	channel Channel
}

func NewRemoteNode(address base.Address, publickey key.Publickey, url string) *RemoteNode {
	return &RemoteNode{
		BaseNodeV0: base.NewBaseNodeV0(address, publickey, url),
	}
}

func (ln *RemoteNode) Channel() Channel {
	ln.RLock()
	defer ln.RUnlock()

	return ln.channel
}

func (ln *RemoteNode) SetChannel(channel Channel) *RemoteNode {
	ln.Lock()
	defer ln.Unlock()

	ln.channel = channel

	return ln
}
