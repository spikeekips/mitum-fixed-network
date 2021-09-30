package states

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/network/discovery/memberlist"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type DiscoveryJoiner struct {
	sync.Mutex
	*logging.Logging
	nodepool         *network.Nodepool
	suffrage         base.Suffrage
	dis              *memberlist.Discovery
	cis              *util.LockedItem
	joinfunc         func() error
	leaveFunc        func(time.Duration) error
	ij               *util.LockedItem
	keeptryingCancel context.CancelFunc
}

func NewDiscoveryJoiner(
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	dis *memberlist.Discovery,
	cis []network.ConnInfo,
) (*DiscoveryJoiner, error) {
	sd := &DiscoveryJoiner{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "states-discovery")
		}),
		nodepool:         nodepool,
		suffrage:         suffrage,
		dis:              dis,
		cis:              util.NewLockedItem(nil),
		ij:               util.NewLockedItem(false),
		keeptryingCancel: func() {},
	}

	return sd, sd.updateURLs(cis...)
}

func (sd *DiscoveryJoiner) Join(maxretry int) error {
	sd.Lock()
	defer sd.Unlock()

	if err := sd.updateURLs(); err != nil {
		return err
	}

	sd.Log().Debug().Msg("trying to join discovery")

	if sd.joinfunc != nil {
		return sd.joinfunc()
	}

	if err := sd.join(maxretry); err != nil {
		sd.Log().Error().Err(err).Msg("failed to join discovery")

		return err
	}

	_ = sd.ij.Set(true)

	sd.Log().Debug().Msg("joined discovery")

	return nil
}

func (sd *DiscoveryJoiner) Leave(timeout time.Duration) error {
	sd.Lock()
	defer sd.Unlock()

	sd.Log().Debug().Msg("trying to leave discovery")
	sd.keeptryingCancel()

	if !sd.IsJoined() {
		return nil
	}

	if sd.leaveFunc != nil {
		return sd.leaveFunc(timeout)
	}

	if err := sd.dis.Leave(timeout); err != nil {
		return err
	}

	_ = sd.ij.Set(false)
	sd.Log().Debug().Msg("left from discovery")

	return nil
}

func (sd *DiscoveryJoiner) KeepTrying(ctx context.Context, ch chan error) {
	if sd.IsJoined() {
		if ch != nil {
			ch <- nil
		}

		return
	}

	sd.Lock()
	defer sd.Unlock()

	sd.keeptryingCancel()

	ctx, sd.keeptryingCancel = context.WithCancel(ctx)

	go func() {
		defer sd.keeptryingCancel()

		if err := sd.keepTrying(ctx); ch != nil {
			ch <- err
		}
	}()
}

func (sd *DiscoveryJoiner) IsJoined() bool {
	return sd.ij.Value().(bool)
}

func (sd *DiscoveryJoiner) updateURLs(cis ...network.ConnInfo) error {
	founds := map[string]memberlist.ConnInfo{}

	existings := sd.urls()
	for i := range existings {
		ci := existings[i]
		founds[ci.String()] = ci
	}

	localci := sd.nodepool.LocalChannel().ConnInfo()

	sd.nodepool.TraverseAliveRemotes(func(_ base.Node, ch network.Channel) bool {
		ci, ok := ch.ConnInfo().(network.HTTPConnInfo)
		if !ok {
			return true
		}

		k := ci.String()
		if _, found := founds[k]; found {
			return true
		}

		founds[k] = memberlist.NewConnInfoWithConnInfo("", ci)

		return true
	})

	for i := range cis {
		ci, ok := cis[i].(network.HTTPConnInfo)
		if !ok {
			return errors.Errorf("discovery conninfo should be network.HTTPConnInfo, not %T", cis[i])
		}

		if localci.URL().String() == ci.URL().String() {
			return errors.Errorf("local conninfo found in discovery urls")
		}

		if _, found := founds[ci.String()]; found {
			continue
		}

		founds[ci.String()] = memberlist.NewConnInfoWithConnInfo("", ci)
	}

	var mcis []memberlist.ConnInfo
	if len(founds) > 0 {
		mcis = make([]memberlist.ConnInfo, len(founds))

		var i int
		for k := range founds {
			mcis[i] = founds[k]
			i++
		}
	}

	_ = sd.cis.Set(mcis)

	return nil
}

func (sd *DiscoveryJoiner) urls() []memberlist.ConnInfo {
	i := sd.cis.Value()
	if i == nil {
		return nil
	}

	return i.([]memberlist.ConnInfo)
}

func (sd *DiscoveryJoiner) keepTrying(ctx context.Context) error {
	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()

end:
	for {
		select {
		case <-ctx.Done():
			sd.Log().Debug().Err(ctx.Err()).Msg("failed to joined discovery; canceled")

			return ctx.Err()
		case <-ticker.C:
			err := sd.join(-1)
			if err == nil {
				break end
			}

			sd.Log().Error().Err(err).Msg("failed to join discovery; keep retrying")
		}
	}

	sd.Log().Debug().Msg("joined discovery")

	return nil
}

func (sd *DiscoveryJoiner) join(maxretry int) error {
	if !sd.suffrage.IsInside(sd.nodepool.LocalNode().Address()) {
		sd.Log().Debug().Msg("local is not suffrage node; no need to join discovery")

		return nil
	}

	if len(sd.urls()) < 1 {
		return util.IgnoreError.Errorf("empty discovery conninfo")
	}

	if err := sd.dis.Join(sd.urls(), maxretry); err != nil {
		return err
	}

	joined := sd.dis.Nodes()
	if len(joined) < 1 {
		return memberlist.JoiningCanceledError.Errorf("failed to join discovery; empty joined nodes")
	}

	var alives []map[string]interface{}
	sd.nodepool.TraverseAliveRemotes(func(no base.Node, ch network.Channel) bool {
		if !sd.suffrage.IsInside(no.Address()) {
			return true
		}

		alives = append(alives, map[string]interface{}{
			no.Address().String(): ch.ConnInfo(),
		})

		return true
	})

	if len(alives) < 1 {
		return memberlist.JoiningCanceledError.Errorf("any nodes did not join, except local")
	}

	return nil
}
