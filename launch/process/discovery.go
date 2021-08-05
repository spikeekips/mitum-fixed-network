package process

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/network/discovery/memberlist"
	"github.com/spikeekips/mitum/util/logging"
)

func JoinDiscovery(
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	dis *memberlist.Discovery,
	cis []memberlist.ConnInfo,
	maxretry int,
	log *logging.Logging,
) error {
	// NOTE join network
	if err := dis.Join(cis, maxretry); err != nil {
		return err
	}

	joined := dis.Nodes()
	if len(joined) < 1 {
		return memberlist.JoiningCanceledError.Errorf("failed to join network; empty joined nodes")
	}

	var alives []map[string]interface{}
	nodepool.TraverseAliveRemotes(func(no base.Node, ch network.Channel) bool {
		if !suffrage.IsInside(no.Address()) {
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

	log.Log().Debug().Interface("joined", alives).Msg("joined network")

	return nil
}

func KeepDiscoveryJoining(
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	dis *memberlist.Discovery,
	cis []memberlist.ConnInfo,
	log *logging.Logging,
) {
	for {
		err := JoinDiscovery(nodepool, suffrage, dis, cis, -1, log)
		if err == nil {
			break
		}

		log.Log().Error().Err(err).Msg("failed to join network; keep retrying")

		<-time.After(time.Second * 2)
	}
}
