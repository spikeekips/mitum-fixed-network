package process

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
)

type testConfigChecker struct {
	suite.Suite
}

func (t *testConfigChecker) ps(ctx context.Context) *pm.Processes {
	ps := pm.NewProcesses().SetContext(ctx)

	processors := []pm.Process{
		ProcessorEncoders,
		ProcessorConfig,
	}

	for i := range processors {
		t.NoError(ps.AddProcess(processors[i], false))
	}

	t.NoError(ps.AddHook(
		pm.HookPrefixPost, ProcessNameEncoders,
		HookNameAddHinters, HookAddHinters(DefaultHinters),
		true,
	))

	return ps
}

func (t *testConfigChecker) TestEmptyLocalNetwork() {
	y := `
network:
`
	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextValueConfigSource, []byte(y))
	ctx = context.WithValue(ctx, ContextValueConfigSourceType, "yaml")

	ps := t.ps(ctx)
	t.NoError(ps.Run())

	cc, err := config.NewChecker(ps.Context())
	t.NoError(err)
	_, err = cc.CheckLocalNetwork()
	t.NoError(err)

	var conf config.LocalNode
	t.NoError(config.LoadConfigContextValue(ps.Context(), &conf))

	t.NotNil(conf.Network())
	t.Equal(config.DefaultLocalNetworkURL, conf.Network().URL())
	t.Equal(config.DefaultLocalNetworkBind, conf.Network().Bind())
}

func (t *testConfigChecker) TestLocalNetwork() {
	{
		y := `
network:
  url: quic://local:54323
  bind: quic://local:54324
`
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextValueConfigSource, []byte(y))
		ctx = context.WithValue(ctx, ContextValueConfigSourceType, "yaml")

		ps := t.ps(ctx)
		t.NoError(ps.Run())

		cc, err := config.NewChecker(ps.Context())
		t.NoError(err)
		_, err = cc.CheckLocalNetwork()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ps.Context(), &conf))

		t.NotNil(conf.Network())
		t.Equal("quic://local:54323", conf.Network().URL().String())
		t.Equal("quic://local:54324", conf.Network().Bind().String())
	}

	{
		y := `
network:
  url: quic://local:54323
`
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextValueConfigSource, []byte(y))
		ctx = context.WithValue(ctx, ContextValueConfigSourceType, "yaml")

		ps := t.ps(ctx)
		t.NoError(ps.Run())

		cc, err := config.NewChecker(ps.Context())
		t.NoError(err)
		_, err = cc.CheckLocalNetwork()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ps.Context(), &conf))

		t.NotNil(conf.Network())
		t.Equal("quic://local:54323", conf.Network().URL().String())
		t.Equal(config.DefaultLocalNetworkBind, conf.Network().Bind())
	}
}

func (t *testConfigChecker) TestEmptyStorage() {
	y := `
storage:
`
	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextValueConfigSource, []byte(y))
	ctx = context.WithValue(ctx, ContextValueConfigSourceType, "yaml")

	ps := t.ps(ctx)
	t.NoError(ps.Run())

	cc, err := config.NewChecker(ps.Context())
	t.NoError(err)
	_, err = cc.CheckStorage()
	t.NoError(err)

	var conf config.LocalNode
	t.NoError(config.LoadConfigContextValue(ps.Context(), &conf))

	t.NotNil(conf.Storage())
	t.Equal(config.DefaultBlockDataPath, conf.Storage().BlockData().Path())
	t.Equal(config.DefaultDatabaseURI, conf.Storage().Database().URI().String())
	t.Equal(config.DefaultDatabaseCache, conf.Storage().Database().Cache().String())
}

func (t *testConfigChecker) TestStorage() {
	{
		y := `
storage:
  blockdata:
    path: "/a/b/c/d"
`
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextValueConfigSource, []byte(y))
		ctx = context.WithValue(ctx, ContextValueConfigSourceType, "yaml")

		ps := t.ps(ctx)
		t.NoError(ps.Run())

		cc, err := config.NewChecker(ps.Context())
		t.NoError(err)
		_, err = cc.CheckStorage()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ps.Context(), &conf))

		t.NotNil(conf.Storage())
		t.Equal("/a/b/c/d", conf.Storage().BlockData().Path())
		t.Equal(config.DefaultDatabaseURI, conf.Storage().Database().URI().String())
		t.Equal(config.DefaultDatabaseCache, conf.Storage().Database().Cache().String())
	}
	{
		y := `
storage:
  blockdata:
    path: "/a/b/c/d"
  database:
    uri: mongodb://1.2.3.4:123456?a=b
`
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextValueConfigSource, []byte(y))
		ctx = context.WithValue(ctx, ContextValueConfigSourceType, "yaml")

		ps := t.ps(ctx)
		t.NoError(ps.Run())

		cc, err := config.NewChecker(ps.Context())
		t.NoError(err)
		_, err = cc.CheckStorage()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ps.Context(), &conf))

		t.NotNil(conf.Storage())
		t.Equal("/a/b/c/d", conf.Storage().BlockData().Path())
		t.Equal("mongodb://1.2.3.4:123456?a=b", conf.Storage().Database().URI().String())
		t.Equal(config.DefaultDatabaseCache, conf.Storage().Database().Cache().String())
	}
	{
		y := `
storage:
  blockdata:
    path: "/a/b/c/d"
  database:
    uri: mongodb://1.2.3.4:123456
    cache: dummy://
`
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextValueConfigSource, []byte(y))
		ctx = context.WithValue(ctx, ContextValueConfigSourceType, "yaml")

		ps := t.ps(ctx)
		t.NoError(ps.Run())

		cc, err := config.NewChecker(ps.Context())
		t.NoError(err)
		_, err = cc.CheckStorage()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ps.Context(), &conf))

		t.NotNil(conf.Storage())
		t.Equal("/a/b/c/d", conf.Storage().BlockData().Path())
		t.Equal("mongodb://1.2.3.4:123456", conf.Storage().Database().URI().String())
		t.Equal("dummy:", conf.Storage().Database().Cache().String())
	}
}

func (t *testConfigChecker) TestPolicyThreshold() {
	{
		y := `
policy:
`
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextValueConfigSource, []byte(y))
		ctx = context.WithValue(ctx, ContextValueConfigSourceType, "yaml")

		ps := t.ps(ctx)
		t.NoError(ps.Run())

		cc, err := config.NewChecker(ps.Context())
		t.NoError(err)
		_, err = cc.CheckPolicy()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ps.Context(), &conf))

		t.Equal(isaac.DefaultPolicyThresholdRatio, conf.Policy().ThresholdRatio())
	}

	{
		y := `
policy:
  threshold: 88.8
`
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextValueConfigSource, []byte(y))
		ctx = context.WithValue(ctx, ContextValueConfigSourceType, "yaml")

		ps := t.ps(ctx)
		t.NoError(ps.Run())

		cc, err := config.NewChecker(ps.Context())
		t.NoError(err)
		_, err = cc.CheckPolicy()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ps.Context(), &conf))

		t.Equal(base.ThresholdRatio(88.8), conf.Policy().ThresholdRatio())
	}
}

func (t *testConfigChecker) TestPolicyMaxOperationsInSeal() {
	{
		y := `
policy:
`
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextValueConfigSource, []byte(y))
		ctx = context.WithValue(ctx, ContextValueConfigSourceType, "yaml")

		ps := t.ps(ctx)
		t.NoError(ps.Run())

		cc, err := config.NewChecker(ps.Context())
		t.NoError(err)
		_, err = cc.CheckPolicy()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ps.Context(), &conf))

		t.Equal(isaac.DefaultPolicyWaitBroadcastingACCEPTBallot, conf.Policy().WaitBroadcastingACCEPTBallot())
	}

	{
		y := `
policy:
  interval-broadcasting-init-ballot: 33s
`
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextValueConfigSource, []byte(y))
		ctx = context.WithValue(ctx, ContextValueConfigSourceType, "yaml")

		ps := t.ps(ctx)
		t.NoError(ps.Run())

		cc, err := config.NewChecker(ps.Context())
		t.NoError(err)
		_, err = cc.CheckPolicy()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ps.Context(), &conf))

		t.Equal(time.Second*33, conf.Policy().IntervalBroadcastingINITBallot())
	}
}

func (t *testConfigChecker) TestTimeoutProcessProposal() {
	{
		y := `
policy:
`
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextValueConfigSource, []byte(y))
		ctx = context.WithValue(ctx, ContextValueConfigSourceType, "yaml")

		ps := t.ps(ctx)
		t.NoError(ps.Run())

		cc, err := config.NewChecker(ps.Context())
		t.NoError(err)
		_, err = cc.CheckPolicy()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ps.Context(), &conf))

		t.Equal(isaac.DefaultPolicyTimeoutProcessProposal, conf.Policy().TimeoutProcessProposal())
	}

	{
		y := `
policy:
  timeout-process-proposal: 33s
`
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextValueConfigSource, []byte(y))
		ctx = context.WithValue(ctx, ContextValueConfigSourceType, "yaml")

		ps := t.ps(ctx)
		t.NoError(ps.Run())

		cc, err := config.NewChecker(ps.Context())
		t.NoError(err)
		_, err = cc.CheckPolicy()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ps.Context(), &conf))

		t.Equal(time.Second*33, conf.Policy().TimeoutProcessProposal())
	}
}

func TestConfigChecker(t *testing.T) {
	suite.Run(t, new(testConfigChecker))
}
