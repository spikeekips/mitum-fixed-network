package contest_module

import (
	"context"
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/node"
	"github.com/spikeekips/mitum/seal"
)

type ChannelNetworkSealHandler func(seal.Seal) (seal.Seal, error)

type ChannelNetwork struct {
	*common.ReaderDaemon
	home    node.Home
	chans   *sync.Map
	handler ChannelNetworkSealHandler
}

func NewChannelNetwork(home node.Home, handler ChannelNetworkSealHandler) *ChannelNetwork {
	cn := &ChannelNetwork{
		ReaderDaemon: common.NewReaderDaemon(false, 0, nil),
		home:         home,
		handler:      handler,
		chans:        &sync.Map{},
	}
	cn.ReaderDaemon.Logger = common.NewLogger(func(c zerolog.Context) zerolog.Context {
		return c.Str("module", "channel-suffrage-network")
	})
	cn.chans.Store(home.Address(), cn)

	return cn
}

func (cn *ChannelNetwork) Home() node.Home {
	return cn.home
}

func (cn *ChannelNetwork) AddMembers(chans ...*ChannelNetwork) *ChannelNetwork {
	for _, ch := range chans {
		if ch.Home().Equal(cn.Home()) {
			continue
		}

		cn.chans.Store(ch.Home().Address(), ch)
	}

	return cn
}

func (cn *ChannelNetwork) Chans() []*ChannelNetwork {
	var chans []*ChannelNetwork
	cn.chans.Range(func(k, v interface{}) bool {
		chans = append(chans, v.(*ChannelNetwork))
		return true
	})

	return chans
}

func (cn *ChannelNetwork) Broadcast(sl seal.Seal) error {
	started := time.Now()

	var wg sync.WaitGroup
	cn.chans.Range(func(_, _ interface{}) bool {
		wg.Add(1)
		return true
	})

	var targets []node.Address
	for _, ch := range cn.Chans() {
		targets = append(targets, ch.Home().Address())
		go func(ch *ChannelNetwork, sl seal.Seal) {
			if !ch.Write(sl) {
				cn.Log().Error().
					Object("to", ch.Home().Address()).
					Object("seal", sl).
					Msg("failed to send seal")
			}
			wg.Done()
		}(ch, sl)
	}

	wg.Wait()

	if cn.Log().Debug().Enabled() {
		tas := zerolog.Arr()
		for _, t := range targets {
			tas.Object(t)
		}

		cn.Log().Debug().
			Object("seal", sl).
			Array("targets", tas).
			Dur("elapsed", time.Now().Sub(started)).
			Msgf("seal sent; %v", sl.Type())
	}

	return nil
}

func (cn *ChannelNetwork) Request(ctx context.Context, n node.Address, sl seal.Seal) (seal.Seal, error) {
	var ch *ChannelNetwork
	if i, found := cn.chans.Load(n); !found {
		return nil, xerrors.Errorf("unknown node; node=%q", n)
	} else {
		ch = i.(*ChannelNetwork)
	}

	return ch.request(ctx, ch, sl)
}

func (cn *ChannelNetwork) request(_ context.Context, ch *ChannelNetwork, sl seal.Seal) (seal.Seal, error) {
	if ch.handler == nil {
		return nil, xerrors.Errorf("node=%q handler not registered", ch.home.Address())
	}

	return ch.handler(sl)
}

func (cn *ChannelNetwork) RequestAll(ctx context.Context, sl seal.Seal) (map[node.Address]seal.Seal, error) {
	results := map[node.Address]seal.Seal{}

	for _, ch := range cn.Chans() {
		r, err := cn.request(ctx, ch, sl)
		if err != nil {
			cn.Log().Error().Err(err).Object("target", ch.home.Address()).Msg("failed to request")
		}
		results[ch.home.Address()] = r
	}

	return results, nil
}
