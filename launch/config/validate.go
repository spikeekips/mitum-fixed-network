package config

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/util/logging"
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
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "config-validator")
		}),
		ctx:    ctx,
		config: conf,
	}

	var l *logging.Logging
	if err := LoadLogContextValue(ctx, &l); err == nil {
		_ = va.SetLogging(l)
	}

	return va, nil
}

func (va *validator) Context() context.Context {
	return va.ctx
}

func (va *validator) CheckNodeAddress() (bool, error) {
	if va.config.Address() == nil {
		return false, errors.Errorf("node address is missing")
	} else if err := va.config.Address().IsValid(nil); err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (va *validator) CheckNodePrivatekey() (bool, error) {
	if va.config.Privatekey() == nil {
		return false, errors.Errorf("node privatekey is missing")
	} else if err := va.config.Privatekey().IsValid(nil); err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (va *validator) CheckNetworkID() (bool, error) {
	if len(va.config.NetworkID()) < 1 {
		return false, errors.Errorf("network id is missing")
	}
	return true, nil
}

func (va *validator) CheckLocalNetwork() (bool, error) {
	conf := va.config.Network()
	if conf == nil {
		return false, errors.Errorf("network is missing")
	}

	if len(conf.Certs()) < 1 {
		return false, errors.Errorf("certificates missing")
	}

	if conf.ConnInfo() == nil {
		return false, errors.Errorf("network url is missing")
	}

	if s := conf.ConnInfo().URL().Scheme; s != "https" {
		return false, errors.Errorf("at this time, publish url only HTTPS allowed, not %q", s)
	}

	if conf.Bind() == nil {
		return false, errors.Errorf("network bind is missing")
	}

	if s := conf.Bind().Scheme; s != "https" {
		return false, errors.Errorf("at this time, bind url only HTTPS allowed, not %q", s)
	}

	return true, nil
}

func (va *validator) CheckStorage() (bool, error) {
	conf := va.config.Storage()
	if conf == nil {
		return false, errors.Errorf("storage is missing")
	}

	if len(conf.Blockdata().Path()) < 1 {
		return false, errors.Errorf("blockdata path is missing")
	}

	if conf.Database().URI() == nil {
		return false, errors.Errorf("database uri is missing")
	}
	if conf.Database().Cache() == nil {
		return false, errors.Errorf("database cache is missing")
	}

	return true, nil
}

func (va *validator) CheckPolicy() (bool, error) {
	conf := va.config.Policy()

	if conf.ThresholdRatio() < 1 {
		return false, errors.Errorf("threshold is zero")
	}

	if conf.MaxOperationsInSeal() < 1 {
		return false, errors.Errorf("max-operations-in-seal is zero")
	}

	if conf.MaxOperationsInProposal() < 1 {
		return false, errors.Errorf("max-operations-in-proposal is zero")
	}

	if conf.TimeoutWaitingProposal() < 1 {
		return false, errors.Errorf("timeout-waiting-proposal is zero")
	}

	if conf.IntervalBroadcastingINITBallot() == 0 {
		return false, errors.Errorf("interval-broadcasting-init-ballot is zero")
	}

	if conf.IntervalBroadcastingProposal() == 0 {
		return false, errors.Errorf("interval-broadcasting-proposal is zero")
	}

	if conf.WaitBroadcastingACCEPTBallot() == 0 {
		return false, errors.Errorf("wait-broadcasting-accept-ballot is zero")
	}

	if conf.IntervalBroadcastingACCEPTBallot() == 0 {
		return false, errors.Errorf("interval-broadcasting-accept-ballot is zero")
	}

	if conf.TimespanValidBallot() == 0 {
		return false, errors.Errorf("timespan-valid-ballot is zero")
	}

	if conf.NetworkConnectionTimeout() == 0 {
		return false, errors.Errorf("network-connection-timeout is zero")
	}

	return true, nil
}

func (va *validator) CheckNodes() (bool, error) {
	if va.config.Address() == nil {
		return false, errors.Errorf("missing local address")
	}

	nodes := va.config.Nodes()

	if len(nodes) < 1 {
		return true, nil
	}

	foundAddresses := map[string]struct{}{}
	for i := range nodes {
		node := nodes[i]

		if a := node.Address(); a == nil {
			return false, errors.Errorf("remote node address is missing")
		} else if err := a.IsValid(nil); err != nil {
			return false, errors.Wrap(err, "invalid remote node address")
		} else if a.Equal(va.config.Address()) {
			return false, errors.Errorf("same address found with local node")
		} else if _, found := foundAddresses[a.String()]; found {
			return false, errors.Errorf("duplicated address found, %s", a)
		} else {
			foundAddresses[a.String()] = struct{}{}
		}

		if node.Publickey() == nil {
			return false, errors.Errorf("publickey of remote node is missing")
		} else if err := node.Publickey().IsValid(nil); err != nil {
			return false, errors.Wrap(err, "invalid remote node publickey")
		}
	}

	return true, nil
}

func (va *validator) CheckSuffrage() (bool, error) {
	if va.config.Address() == nil {
		return false, errors.Errorf("missing local address")
	}

	conf := va.config.Suffrage()
	if conf == nil {
		return false, errors.Errorf("suffrage is missing")
	}

	if err := conf.IsValid(nil); err != nil {
		return false, err
	}

	nodes := va.config.Nodes()
	if len(conf.Nodes()) < 1 {
		if len(nodes) < 1 {
			return false, errors.Errorf("suffrage nodes and nodes both empty")
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
			return false, errors.Errorf("node, %q in suffrage not found in nodes", n)
		}
	}

	return true, nil
}

func (va *validator) CheckProposalProcessor() (bool, error) {
	conf := va.config.ProposalProcessor()
	if conf == nil {
		return false, errors.Errorf("proposal_processor is missing")
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
			return false, errors.Errorf("nil operation found")
		} else if err := op.IsValid(va.config.NetworkID()); err != nil {
			return false, errors.Wrap(err, "invalid operation found")
		}
	}

	return true, nil
}

func (va *validator) CheckLocalConfig() (bool, error) {
	conf := va.config.LocalConfig()

	switch t := conf.SyncInterval(); {
	case t < 1:
		return false, errors.Errorf("empty sync-interval")
	case t < time.Second:
		return false, errors.Errorf("sync-interval too narrow, %q", t)
	default:
		return true, nil
	}
}
