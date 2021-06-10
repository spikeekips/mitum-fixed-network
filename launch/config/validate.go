package config

import (
	"context"
	"time"

	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type validator struct {
	*logging.Logging
	ctx    context.Context
	config LocalNode
}

func NewValidator(ctx context.Context) (*validator, error) { // revive:disable-line:unexported-return
	var conf LocalNode
	if err := LoadConfigContextValue(ctx, &conf); err != nil {
		return nil, err
	}

	va := &validator{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "config-validator")
		}),
		ctx:    ctx,
		config: conf,
	}

	var l logging.Logger
	if err := LoadLogContextValue(ctx, &l); err == nil {
		_ = va.SetLogger(l)
	}

	return va, nil
}

func (va *validator) Context() context.Context {
	return va.ctx
}

func (va *validator) CheckNodeAddress() (bool, error) {
	if va.config.Address() == nil {
		return false, xerrors.Errorf("node address is missing")
	} else if err := va.config.Address().IsValid(nil); err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (va *validator) CheckNodePrivatekey() (bool, error) {
	if va.config.Privatekey() == nil {
		return false, xerrors.Errorf("node privatekey is missing")
	} else if err := va.config.Privatekey().IsValid(nil); err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (va *validator) CheckNetworkID() (bool, error) {
	if len(va.config.NetworkID()) < 1 {
		return false, xerrors.Errorf("network id is missing")
	}
	return true, nil
}

func (va *validator) CheckLocalNetwork() (bool, error) {
	conf := va.config.Network()
	if conf == nil {
		return false, xerrors.Errorf("network is missing")
	}

	if conf.URL() == nil {
		return false, xerrors.Errorf("network url is missing")
	}

	if conf.Bind() == nil {
		return false, xerrors.Errorf("network bind is missing")
	}

	return true, nil
}

func (va *validator) CheckStorage() (bool, error) {
	conf := va.config.Storage()
	if conf == nil {
		return false, xerrors.Errorf("storage is missing")
	}

	if len(conf.BlockData().Path()) < 1 {
		return false, xerrors.Errorf("blockdata path is missing")
	}

	if conf.Database().URI() == nil {
		return false, xerrors.Errorf("database uri is missing")
	}
	if conf.Database().Cache() == nil {
		return false, xerrors.Errorf("database cache is missing")
	}

	return true, nil
}

func (va *validator) CheckPolicy() (bool, error) {
	conf := va.config.Policy()

	if conf.ThresholdRatio() < 1 {
		return false, xerrors.Errorf("threshold is zero")
	}

	if conf.MaxOperationsInSeal() < 1 {
		return false, xerrors.Errorf("max-operations-in-seal is zero")
	}

	if conf.MaxOperationsInProposal() < 1 {
		return false, xerrors.Errorf("max-operations-in-proposal is zero")
	}

	if conf.TimeoutWaitingProposal() < 1 {
		return false, xerrors.Errorf("timeout-waiting-proposal is zero")
	}

	if conf.IntervalBroadcastingINITBallot() == 0 {
		return false, xerrors.Errorf("interval-broadcasting-init-ballot is zero")
	}

	if conf.IntervalBroadcastingProposal() == 0 {
		return false, xerrors.Errorf("interval-broadcasting-proposal is zero")
	}

	if conf.WaitBroadcastingACCEPTBallot() == 0 {
		return false, xerrors.Errorf("wait-broadcasting-accept-ballot is zero")
	}

	if conf.IntervalBroadcastingACCEPTBallot() == 0 {
		return false, xerrors.Errorf("interval-broadcasting-accept-ballot is zero")
	}

	if conf.TimespanValidBallot() == 0 {
		return false, xerrors.Errorf("timespan-valid-ballot is zero")
	}

	if conf.NetworkConnectionTimeout() == 0 {
		return false, xerrors.Errorf("network-connection-timeout is zero")
	}

	return true, nil
}

func (va *validator) CheckNodes() (bool, error) {
	if va.config.Address() == nil {
		return false, xerrors.Errorf("missing local address")
	}

	nodes := va.config.Nodes()

	if len(nodes) < 1 {
		return true, nil
	}

	foundAddresses := map[string]struct{}{}
	foundNetworks := map[string]struct{}{}
	for i := range nodes {
		node := nodes[i]

		if a := node.Address(); a == nil {
			return false, xerrors.Errorf("remote node address is missing")
		} else if err := a.IsValid(nil); err != nil {
			return false, xerrors.Errorf("invalid remote node address: %w", err)
		} else if a.Equal(va.config.Address()) {
			return false, xerrors.Errorf("same address found with local node")
		} else if _, found := foundAddresses[a.String()]; found {
			return false, xerrors.Errorf("duplicated address found, %s", a)
		} else {
			foundAddresses[a.String()] = struct{}{}
		}

		if u := node.URL(); u == nil {
			return false, xerrors.Errorf("network of remote node is missing")
		} else if u.String() == va.config.Network().URL().String() {
			return false, xerrors.Errorf("same network found with local node")
		} else if _, found := foundNetworks[u.String()]; found {
			return false, xerrors.Errorf("duplicated network found, %s", u)
		} else {
			foundNetworks[u.String()] = struct{}{}
		}

		if node.Publickey() == nil {
			return false, xerrors.Errorf("publickey of remote node is missing")
		} else if err := node.Publickey().IsValid(nil); err != nil {
			return false, xerrors.Errorf("invalid remote node publickey: %w", err)
		}
	}

	return true, nil
}

func (va *validator) CheckSuffrage() (bool, error) {
	if va.config.Address() == nil {
		return false, xerrors.Errorf("missing local address")
	}

	conf := va.config.Suffrage()
	if conf == nil {
		return false, xerrors.Errorf("suffrage is missing")
	}

	if err := conf.IsValid(nil); err != nil {
		return false, err
	}

	nodes := va.config.Nodes()
	if len(conf.Nodes()) < 1 {
		if len(nodes) < 1 {
			return false, xerrors.Errorf("suffrage nodes and nodes both empty")
		}

		return true, nil
	}

	for i := range conf.Nodes() {
		n := conf.Nodes()[i]

		var found bool
		if n.Equal(va.config.Address()) {
			found = true
		} else {
			for j := range nodes {
				if n.Equal(nodes[j].Address()) {
					found = true

					break
				}
			}
		}

		if !found {
			return false, xerrors.Errorf("node, %q in suffrage not found in nodes", n)
		}
	}

	return true, nil
}

func (va *validator) CheckProposalProcessor() (bool, error) {
	conf := va.config.ProposalProcessor()
	if conf == nil {
		return false, xerrors.Errorf("proposal_processor is missing")
	}

	return true, nil
}

func (va *validator) CheckGenesisOperations() (bool, error) {
	ops := va.config.GenesisOperations()
	if len(ops) < 1 {
		return true, nil
	}

	for i := range ops {
		if op := ops[i]; op == nil {
			return false, xerrors.Errorf("nil operation found")
		} else if err := op.IsValid(va.config.NetworkID()); err != nil {
			return false, xerrors.Errorf("invalid operation found: %w", err)
		}
	}

	return true, nil
}

func (va *validator) CheckLocalConfig() (bool, error) {
	conf := va.config.LocalConfig()

	switch t := conf.SyncInterval(); {
	case t < 1:
		return false, xerrors.Errorf("empty sync-interval")
	case t < time.Second:
		return false, xerrors.Errorf("sync-interval too narrow, %q", t)
	default:
		return true, nil
	}
}
