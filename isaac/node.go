package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/key"
)

type Node interface {
	Address() Address
	Publickey() key.Publickey
	Channel() NetworkChannel
}

type LocalNode struct {
	sync.RWMutex
	address    Address
	publickey  key.Publickey
	privatekey key.Privatekey
	channel    NetworkChannel
}

func NewLocalNode(address Address, privatekey key.Privatekey) *LocalNode {
	return &LocalNode{
		address:    address,
		publickey:  privatekey.Publickey(),
		privatekey: privatekey,
	}
}

func (ln *LocalNode) Address() Address {
	return ln.address
}

func (ln *LocalNode) Publickey() key.Publickey {
	ln.RLock()
	defer ln.RUnlock()

	return ln.publickey
}

func (ln *LocalNode) Privatekey() key.Privatekey {
	ln.RLock()
	defer ln.RUnlock()

	return ln.privatekey
}

func (ln *LocalNode) Channel() NetworkChannel {
	ln.RLock()
	defer ln.RUnlock()

	return ln.channel
}

func (ln *LocalNode) SetChannel(channel NetworkChannel) *LocalNode {
	ln.Lock()
	defer ln.Unlock()

	ln.channel = channel

	return ln
}

type RemoteNode struct {
	sync.RWMutex
	address   Address
	publickey key.Publickey
	channel   NetworkChannel
}

func NewRemoteNode(address Address, publickey key.Publickey) *RemoteNode {
	return &RemoteNode{
		address:   address,
		publickey: publickey,
	}
}

func (ln *RemoteNode) Address() Address {
	return ln.address
}

func (ln *RemoteNode) Publickey() key.Publickey {
	return ln.publickey
}

func (ln *RemoteNode) Channel() NetworkChannel {
	ln.RLock()
	defer ln.RUnlock()

	return ln.channel
}

func (ln *RemoteNode) SetChannel(channel NetworkChannel) *RemoteNode {
	ln.Lock()
	defer ln.Unlock()

	ln.channel = channel

	return ln
}
