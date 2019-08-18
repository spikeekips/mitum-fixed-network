package contest_module

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/node"
	"github.com/spikeekips/mitum/seal"
)

type ChannelNetworkSealHandler func(seal.Seal) (seal.Seal, error)

type ChannelNetwork struct {
	sync.RWMutex
	*common.ReaderDaemon
	home    node.Home
	chans   map[node.Address]*ChannelNetwork
	handler ChannelNetworkSealHandler
}

func NewChannelNetwork(home node.Home, handler ChannelNetworkSealHandler) *ChannelNetwork {
	cn := &ChannelNetwork{
		ReaderDaemon: common.NewReaderDaemon(false, 0, nil),
		home:         home,
		handler:      handler,
		chans:        map[node.Address]*ChannelNetwork{},
	}
	cn.ReaderDaemon.Logger = common.NewLogger(log, "module", "channel-suffrage-network")
	cn.chans[home.Address()] = cn

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

func (cn *ChannelNetwork) Chans() []*ChannelNetwork {
	cn.RLock()
	defer cn.RUnlock()

	var chans []*ChannelNetwork
	for _, ch := range cn.chans {
		chans = append(chans, ch)
	}

	return chans
}

func (cn *ChannelNetwork) Broadcast(sl seal.Seal) error {
	started := time.Now()

	var wg sync.WaitGroup
	wg.Add(len(cn.chans))

	var targets []node.Address
	for _, ch := range cn.Chans() {
		targets = append(targets, ch.Home().Address())
		go func(a node.Address, sl seal.Seal) {
			if !ch.Write(sl) {
				cn.Log().Error("failed to send seal", "to", a, "seal", sl)
			}
			wg.Done()
		}(ch.Home().Address(), sl)
	}

	wg.Wait()

	cn.Log().Debug(
		fmt.Sprintf("seal sent; %v", sl.Type()),
		"seal", sl,
		"targets", targets,
		"elapsed", time.Now().Sub(started),
	)

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
