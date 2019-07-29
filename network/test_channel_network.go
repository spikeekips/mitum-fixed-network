// +build test

package network

import (
	"context"
	"sync"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/node"
	"github.com/spikeekips/mitum/seal"
	"golang.org/x/xerrors"
)

type ChannelNetworkSealHandler func(seal.Seal) (seal.Seal, error)

type ChannelNetwork struct {
	sync.RWMutex
	*common.Logger
	*common.ReaderDaemon
	home    node.Home
	chans   map[node.Address]*ChannelNetwork
	handler ChannelNetworkSealHandler
}

func NewChannelNetwork(home node.Home, handler ChannelNetworkSealHandler) *ChannelNetwork {
	cn := &ChannelNetwork{
		Logger:       common.NewLogger(log, "module", "channel-suffrage-network"),
		ReaderDaemon: common.NewReaderDaemon(false, 0, nil),
		home:         home,
		handler:      handler,
	}
	cn.chans = map[node.Address]*ChannelNetwork{home.Address(): cn}

	return cn
}

func (cn *ChannelNetwork) Home() node.Home {
	return cn.home
}

func (cn *ChannelNetwork) AddMembers(chans ...*ChannelNetwork) *ChannelNetwork {
	cn.Lock()
	defer cn.Unlock()

	for _, ch := range chans {
		if ch.Home().Equal(cn.Home()) {
			continue
		}
		cn.chans[ch.Home().Address()] = ch
	}

	return cn
}

func (cn *ChannelNetwork) SetHandler(handler ChannelNetworkSealHandler) *ChannelNetwork {
	cn.handler = handler
	return cn
}

func (cn *ChannelNetwork) Broadcast(sl seal.Seal) error {
	cn.RLock()
	defer cn.RUnlock()

	var wg sync.WaitGroup
	wg.Add(len(cn.chans))

	for _, ch := range cn.chans {
		go func(ch *ChannelNetwork) {
			defer wg.Done()

			if ch.Write(sl) {
				cn.Log().Debug("sent seal", "to", ch.Home().Address(), "seal", sl)
			} else {
				cn.Log().Error("failed to send seal", "to", ch.Home().Address(), "seal", sl)
			}
		}(ch)
	}

	wg.Wait()

	return nil
}

func (cn *ChannelNetwork) Request(_ context.Context, n node.Address, sl seal.Seal) (seal.Seal, error) {
	cn.RLock()
	defer cn.RUnlock()

	ch, found := cn.chans[n]
	if !found {
		return nil, xerrors.Errorf("unknown node; node=%q", n)
	}

	if ch.handler == nil {
		return nil, xerrors.Errorf("node=%q handler not registered", n)
	}

	return ch.handler(sl)
}

func (cn *ChannelNetwork) RequestAll(ctx context.Context, sl seal.Seal) (map[node.Address]seal.Seal, error) {
	results := map[node.Address]seal.Seal{}

	cn.RLock()
	defer cn.RUnlock()

	for n := range cn.chans {
		r, err := cn.Request(ctx, n, sl)
		if err != nil {
			cn.Log().Error("failed to request", "target", n, "error", err)
		}
		results[n] = r
	}

	return results, nil
}
