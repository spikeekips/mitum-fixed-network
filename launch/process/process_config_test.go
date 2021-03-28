package process

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/util/logging"
)

type testProcessConfig struct {
	suite.Suite
}

func (t *testProcessConfig) pm(ctx context.Context) *pm.Processes {
	ps := pm.NewProcesses().SetContext(ctx)

	processors := []pm.Process{
		ProcessorConfig,
		ProcessorEncoders,
	}

	for _, pr := range processors {
		t.NoError(ps.AddProcess(pr, false))
	}

	t.NoError(ps.AddHook(
		pm.HookPrefixPost, ProcessNameEncoders,
		HookNameAddHinters, HookAddHinters(DefaultHinters),
		true,
	))

	return ps
}

func (t *testProcessConfig) TestSimple() {
	y := `
privatekey: KzmnCUoBrqYbkoP8AUki1AJsyKqxNsiqdrtTB2onyzQfB6MQ5Sef-0112:0.0.1
network-id: show me
`
	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextValueConfigSource, []byte(y))
	ctx = context.WithValue(ctx, ContextValueConfigSourceType, "yaml")

	ps := t.pm(ctx)

	t.NoError(ps.Run())

	var conf config.LocalNode
	err := config.LoadConfigContextValue(ps.Context(), &conf)
	t.NoError(err)

	t.Equal("KzmnCUoBrqYbkoP8AUki1AJsyKqxNsiqdrtTB2onyzQfB6MQ5Sef-0112:0.0.1", conf.Privatekey().String())
	t.Equal([]byte("show me"), conf.NetworkID().Bytes())
}

func TestProcessConfig(t *testing.T) {
	suite.Run(t, new(testProcessConfig))
}

type testConfig struct {
	suite.Suite
}

func (t *testConfig) ready(y string) *pm.Processes {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextValueConfigSource, []byte(y))
	ctx = context.WithValue(ctx, ContextValueConfigSourceType, "yaml")
	ctx = context.WithValue(ctx, ContextValueLog, logging.NilLogger)

	ps := pm.NewProcesses().SetContext(ctx)

	t.NoError(ps.AddProcess(ProcessorEncoders, false))
	t.NoError(ps.AddHook(
		pm.HookPrefixPost, ProcessNameEncoders,
		HookNameAddHinters, HookAddHinters(DefaultHinters),
		true,
	))

	t.NoError(Config(ps))

	return ps
}

func (t *testConfig) TestSimple() {
	y := `
address: n0-010a:0.0.1
privatekey: KzmnCUoBrqYbkoP8AUki1AJsyKqxNsiqdrtTB2onyzQfB6MQ5Sef-0112:0.0.1
network-id: show me
nodes:
  - address: n1-010a:0.0.1
    url: quic://local:54322
    publickey: 27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb-0113:0.0.1
time-server: ""
`

	ps := t.ready(y)
	t.NoError(ps.Run())

	var conf config.LocalNode
	err := config.LoadConfigContextValue(ps.Context(), &conf)
	t.NoError(err)

	t.Equal("n0-010a:0.0.1", conf.Address().String())
	t.Equal("KzmnCUoBrqYbkoP8AUki1AJsyKqxNsiqdrtTB2onyzQfB6MQ5Sef-0112:0.0.1", conf.Privatekey().String())
	t.Equal([]byte("show me"), conf.NetworkID().Bytes())

	t.Equal(1, len(conf.Nodes()))
	t.Equal("n1-010a:0.0.1", conf.Nodes()[0].Address().String())
	t.Equal("quic://local:54322", conf.Nodes()[0].URL().String())
	t.Equal("27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb-0113:0.0.1", conf.Nodes()[0].Publickey().String())

	// check empties
	t.Equal(config.DefaultLocalNetworkURL.String(), conf.Network().URL().String())
	t.Equal(config.DefaultLocalNetworkBind.String(), conf.Network().Bind().String())

	t.Equal(config.DefaultBlockDataPath, conf.Storage().BlockData().Path())
	t.Equal(config.DefaultDatabaseURI, conf.Storage().Database().URI().String())
	t.Equal(config.DefaultDatabaseCache, conf.Storage().Database().Cache().String())

	t.IsType(config.RoundrobinSuffrage{}, conf.Suffrage())
	t.IsType(config.DefaultProposalProcessor{}, conf.ProposalProcessor())

	t.Equal(0, len(conf.GenesisOperations()))

	t.Equal(isaac.DefaultPolicyThresholdRatio, conf.Policy().ThresholdRatio())
	t.Equal(isaac.DefaultPolicyWaitBroadcastingACCEPTBallot, conf.Policy().WaitBroadcastingACCEPTBallot())
	t.Equal(isaac.DefaultPolicyTimeoutProcessProposal, conf.Policy().TimeoutProcessProposal())
	t.Empty(conf.TimeServer())
}

func (t *testConfig) TestInValidSuffrage() {
	y := `
address: n0-010a:0.0.1
privatekey: KzmnCUoBrqYbkoP8AUki1AJsyKqxNsiqdrtTB2onyzQfB6MQ5Sef-0112:0.0.1
network-id: show me
nodes:
  - address: n1-010a:0.0.1
    url: quic://local:54322
    publickey: 27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb-0113:0.0.1
suffrage:
  type: show-me
`

	ps := t.ready(y)
	err := ps.Run()
	t.Contains(err.Error(), "unknown suffrage found")
}

func TestConfig(t *testing.T) {
	suite.Run(t, new(testConfig))
}
