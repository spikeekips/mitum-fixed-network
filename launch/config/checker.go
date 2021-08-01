package config

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"time"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type checker struct {
	*logging.Logging
	ctx    context.Context
	config LocalNode
}

func NewChecker(ctx context.Context) (*checker, error) { // revive:disable-line:unexported-return
	var conf LocalNode
	if err := LoadConfigContextValue(ctx, &conf); err != nil {
		return nil, err
	}

	cc := &checker{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "config-checker")
		}),
		ctx:    ctx,
		config: conf,
	}

	var l logging.Logger
	if err := LoadLogContextValue(ctx, &l); err == nil {
		_ = cc.SetLogger(l)
	}

	return cc, nil
}

func (cc *checker) Context() context.Context {
	return cc.ctx
}

func (cc *checker) CheckLocalNetwork() (bool, error) {
	conf := cc.config.Network()

	insecure, err := cc.checkLocalNetworkCerts(conf)
	if err != nil {
		return false, err
	}

	if err := cc.checkLocalNetworkConnInfo(conf, insecure); err != nil {
		return false, err
	}

	if conf.Bind() == nil {
		if err := conf.SetBind(DefaultLocalNetworkBind.String()); err != nil {
			return false, err
		}
	}

	if conf.Cache() == nil {
		if err := conf.SetCache(DefaultLocalNetworkCache); err != nil {
			return false, err
		}
	}

	if conf.SealCache() == nil {
		if err := conf.SetSealCache(DefaultLocalNetworkSealCache); err != nil {
			return false, err
		}
	}

	if conf.RateLimit() != nil {
		if err := cc.checkRateLimit(); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (cc *checker) CheckStorage() (bool, error) {
	conf := cc.config.Storage()

	if len(conf.BlockData().Path()) < 1 {
		if err := conf.BlockData().SetPath(DefaultBlockDataPath); err != nil {
			return false, err
		}
	}

	if conf.Database().URI() == nil {
		if err := conf.Database().SetURI(DefaultDatabaseURI); err != nil {
			return false, err
		}
	}
	if conf.Database().Cache() == nil {
		if err := conf.Database().SetCache(DefaultDatabaseCache); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (cc *checker) CheckPolicy() (bool, error) {
	conf := cc.config.Policy()

	if conf.ThresholdRatio() == 0 {
		if err := conf.SetThresholdRatio(isaac.DefaultPolicyThresholdRatio.Float64()); err != nil {
			return false, err
		}
	}

	uints := [][3]interface{}{
		{conf.MaxOperationsInSeal(), conf.SetMaxOperationsInSeal, isaac.DefaultPolicyMaxOperationsInSeal},             // revive:disable-line:line-length-limit
		{conf.MaxOperationsInProposal(), conf.SetMaxOperationsInProposal, isaac.DefaultPolicyMaxOperationsInProposal}, // revive:disable-line:line-length-limit
	}

	for i := range uints {
		v := uints[i][0].(uint)
		d := uints[i][2].(uint)
		f := uints[i][1].(func(uint) error)

		if v > 0 {
			continue
		}
		if err := f(d); err != nil {
			return false, err
		}
	}

	durs := [][3]interface{}{
		{conf.TimeoutWaitingProposal(), conf.SetTimeoutWaitingProposal, isaac.DefaultPolicyTimeoutWaitingProposal},                               // revive:disable-line:line-length-limit
		{conf.IntervalBroadcastingINITBallot(), conf.SetIntervalBroadcastingINITBallot, isaac.DefaultPolicyIntervalBroadcastingINITBallot},       // revive:disable-line:line-length-limit
		{conf.IntervalBroadcastingProposal(), conf.SetIntervalBroadcastingProposal, isaac.DefaultPolicyIntervalBroadcastingProposal},             // revive:disable-line:line-length-limit
		{conf.WaitBroadcastingACCEPTBallot(), conf.SetWaitBroadcastingACCEPTBallot, isaac.DefaultPolicyWaitBroadcastingACCEPTBallot},             // revive:disable-line:line-length-limit
		{conf.IntervalBroadcastingACCEPTBallot(), conf.SetIntervalBroadcastingACCEPTBallot, isaac.DefaultPolicyIntervalBroadcastingACCEPTBallot}, // revive:disable-line:line-length-limit
		{conf.TimespanValidBallot(), conf.SetTimespanValidBallot, isaac.DefaultPolicyTimespanValidBallot},                                        // revive:disable-line:line-length-limit
		{conf.NetworkConnectionTimeout(), conf.SetNetworkConnectionTimeout, isaac.DefaultPolicyNetworkConnectionTimeout},                         // revive:disable-line:line-length-limit
	}

	for i := range durs {
		v := durs[i][0].(time.Duration)
		d := durs[i][2].(time.Duration)
		f := durs[i][1].(func(string) error)

		if v > 0 {
			continue
		}
		if err := f(d.String()); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (cc *checker) checkRateLimit() error {
	rcc := NewRateLimitChecker(
		cc.ctx,
		cc.config.Network().RateLimit(),
		map[string]RateLimitRules{
			"suffrage": NewBaseRateLimitRules(DefaultSuffrageRateLimit),
			"world":    NewBaseRateLimitRules(DefaultWorldRateLimit),
		},
	)

	if err := util.NewChecker("config-ratelimit-checker", []util.CheckerFunc{
		rcc.Initialize,
		rcc.Check,
	}).Check(); err != nil {
		if !xerrors.Is(err, util.IgnoreError) {
			return err
		}
	}

	return cc.config.Network().SetRateLimit(rcc.Config())
}

func (cc *checker) checkLocalNetworkCerts(conf LocalNetwork) (bool, error) {
	var host string
	if conf.ConnInfo() == nil {
		host = DefaultLocalNetworkURL.Hostname()
	} else {
		host = conf.ConnInfo().URL().Hostname()
	}

	certs := conf.Certs()
	if len(certs) < 1 {
		if priv, err := util.GenerateED25519Privatekey(); err != nil {
			return false, err
		} else if ct, err := util.GenerateTLSCerts(host, priv); err != nil {
			return false, err
		} else {
			certs = ct
		}
	}

	insecure, err := cc.verifyCerts(host, certs)
	if err != nil {
		return false, err
	}

	return insecure, conf.SetCerts(certs)
}

func (*checker) verifyCerts(host string, certs []tls.Certificate) (bool /* insecure */, error) {
	cert, err := x509.ParseCertificate(certs[0].Certificate[0])
	if err != nil {
		return false, err
	}

	opts := x509.VerifyOptions{
		DNSName: host,
	}

	_, err = cert.Verify(opts)
	if err == nil {
		return false, nil
	}

	var cerr x509.CertificateInvalidError
	if xerrors.As(err, &cerr) {
		return true, nil
	}
	var herr x509.HostnameError
	if xerrors.As(err, &herr) {
		return true, nil
	}
	var uerr x509.UnknownAuthorityError
	if xerrors.As(err, &uerr) {
		return true, nil
	}

	return false, err
}

func (*checker) checkLocalNetworkConnInfo(conf LocalNetwork, insecure bool) error {
	if conf.ConnInfo() == nil {
		return conf.SetConnInfo(network.NewHTTPConnInfo(DefaultLocalNetworkURL, insecure))
	}

	connInfo := conf.ConnInfo()
	if connInfo.Insecure() == insecure {
		return nil
	}

	return conf.SetConnInfo(network.NewHTTPConnInfo(connInfo.URL(), insecure))
}
