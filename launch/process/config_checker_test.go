package process

import (
	"context"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
)

type testConfigChecker struct {
	suite.Suite
	root string
}

func (t *testConfigChecker) SetupSuite() {
	p, err := os.MkdirTemp("", "")
	t.NoError(err)
	t.root = p
}

func (t *testConfigChecker) TearDownSuite() {
	_ = os.RemoveAll(t.root)
}

func (t *testConfigChecker) newCertFiles(host string) (string, string) {
	priv, err := util.GenerateED25519Privatekey()
	t.NoError(err)

	k, c, err := util.GenerateTLSCertsPair(host, priv)
	t.NoError(err)

	var kfile, cfile string

	f, err := ioutil.TempFile(t.root, "")
	t.NoError(err)
	t.NoError(pem.Encode(f, k))
	_ = f.Close()
	kfile = f.Name()

	f, err = ioutil.TempFile(t.root, "")
	t.NoError(err)
	t.NoError(pem.Encode(f, c))
	_ = f.Close()
	cfile = f.Name()

	return kfile, cfile
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
		HookNameAddHinters, HookAddHinters(launch.EncoderTypes, launch.EncoderHinters),
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
	t.Equal(config.DefaultLocalNetworkURL, conf.Network().ConnInfo().URL())
	t.Equal(config.DefaultLocalNetworkBind, conf.Network().Bind())
	t.NotNil(conf.Network().Certs())
	t.True(conf.Network().ConnInfo().Insecure())
}

func (t *testConfigChecker) TestLocalNetwork() {
	{
		y := `
network:
  url: https://localhost:54323
  bind: https://localhost:54324
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
		t.Equal("https://127.0.0.1:54323", conf.Network().ConnInfo().URL().String())
		t.Equal("https://localhost:54324", conf.Network().Bind().String())
		t.Equal(config.DefaultLocalNetworkCache, conf.Network().Cache().String())
		t.Equal(config.DefaultLocalNetworkSealCache, conf.Network().SealCache().String())
		t.NotNil(conf.Network().Certs())
		t.True(conf.Network().ConnInfo().Insecure())
	}

	{
		y := `
network:
  url: https://local:54323
  cache: gcache:?type=lru&size=33&expire=44s
  seal-cache: gcache:?type=lru&size=55&expire=66s
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
		t.Equal("https://local:54323", conf.Network().ConnInfo().URL().String())
		t.Equal(config.DefaultLocalNetworkBind, conf.Network().Bind())
		t.Equal("gcache:?type=lru&size=33&expire=44s", conf.Network().Cache().String())
		t.Equal("gcache:?type=lru&size=55&expire=66s", conf.Network().SealCache().String())
	}
}

func (t *testConfigChecker) TestRateLimitWrongCache() {
	y := `
network:
  url: https://local:54323
  bind: https://local:54324
  rate-limit:
    cache: gcache:?type=lru&size=33&expire=44s
`
	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextValueConfigSource, []byte(y))
	ctx = context.WithValue(ctx, ContextValueConfigSourceType, "yaml")

	ps := t.ps(ctx)
	err := ps.Run()

	t.Contains(err.Error(), "unknown ratelimit cache uri")
}

func (t *testConfigChecker) TestRateLimitCache() {
	y := `
network:
  url: https://local:54323
  bind: https://local:54324
  rate-limit:
    cache: memory:?prefix=showme
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
	t.NotNil(conf.Network().RateLimit())

	t.Equal("memory:?prefix=showme", conf.Network().RateLimit().Cache().String())
}

func (t *testConfigChecker) TestEmptyRateLimit() {
	y := `
network:
  url: https://local:54323
  bind: https://local:54324
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
	t.Nil(conf.Network().RateLimit())
}

func (t *testConfigChecker) TestRateLimit() {
	y := `
network:
  url: https://local:54323
  bind: https://local:54324

  rate-limit:
    preset:
      suffrage:
         blockdata-maps: 700/3m
         blockdata: 80/4m

      world:
         send-seal: 300/2m
         blockdata: 60/1m

      others:
         blockdata: 60/1m

    192.168.3.3:
      preset: world

    192.168.1.0/24:
      preset: suffrage
      send-seal: 222/1s
      blockdata-maps: 333/2s
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
	t.NotNil(conf.Network().RateLimit())

	rc := conf.Network().RateLimit()
	t.Equal(int64(80), rc.Preset()["suffrage"].Rules()["blockdata"].Limit)
	t.Equal(time.Minute*4, rc.Preset()["suffrage"].Rules()["blockdata"].Period)

	// the not defined set from default
	t.Equal(config.DefaultSuffrageRateLimit["send-seal"].Limit, rc.Preset()["suffrage"].Rules()["send-seal"].Limit)
	t.Equal(config.DefaultSuffrageRateLimit["send-seal"].Period, rc.Preset()["suffrage"].Rules()["send-seal"].Period)

	// the not defined set from default world or suffrage
	var found bool
	_, found = rc.Preset()["others"].Rules()["send-seal"]
	t.False(found)
	t.Equal(int64(60), rc.Preset()["others"].Rules()["blockdata"].Limit)
	t.Equal(time.Minute*1, rc.Preset()["others"].Rules()["blockdata"].Period)

	t.Equal("192.168.1.0/24", rc.Rules()[1].Target())
	t.Equal(int64(222), rc.Rules()[1].Rules()["send-seal"].Limit)
	t.Equal(time.Second, rc.Rules()[1].Rules()["send-seal"].Period)
	t.Equal("192.168.1.0/24", rc.Rules()[1].IPNet().String())
}

func (t *testConfigChecker) TestRateLimitEmptyRules() {
	y := `
network:
  url: https://local:54323
  bind: https://local:54324

  rate-limit:
    preset:
      suffrage:
         blockdata-maps: 700/3m
         blockdata: 80/4m

      world:
         send-seal: 300/2m
         blockdata: 60/1m

      others:
         blockdata: 60/1m
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
	t.NotNil(conf.Network().RateLimit())

	rc := conf.Network().RateLimit()

	t.Empty(rc.Rules())
}

func (t *testConfigChecker) TestRateLimitEmptyRule() {
	y := `
network:
  url: https://local:54323
  bind: https://local:54324

  rate-limit:
      192.168.0.1:
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
	t.NotNil(conf.Network().RateLimit())

	rc := conf.Network().RateLimit()

	t.Equal(1, len(rc.Rules()))

	r := rc.Rules()[0]
	t.Equal("192.168.0.1", r.Target())
	t.Equal("", r.Preset())
	t.Empty(r.Rules())
}

func (t *testConfigChecker) TestRateLimitRuleWithoutPreset() {
	y := `
network:
  url: https://local:54323
  bind: https://local:54324

  rate-limit:
    192.168.0.1:
      send-seal: 222/2s
      seals: 333/3s
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
	t.NotNil(conf.Network().RateLimit())

	rc := conf.Network().RateLimit()

	t.Equal(1, len(rc.Rules()))

	r := rc.Rules()[0]
	t.Equal("192.168.0.1", r.Target())
	t.Equal("", r.Preset())
	t.Equal(2, len(r.Rules()))

	t.Equal(int64(222), r.Rules()["send-seal"].Limit)
	t.Equal(time.Second*2, r.Rules()["send-seal"].Period)
	t.Equal(int64(333), r.Rules()["seals"].Limit)
	t.Equal(time.Second*3, r.Rules()["seals"].Period)
}

func (t *testConfigChecker) TestLocalNetworkEmptyCertificates() {
	{
		y := `
network:
  url: https://local:54323
  bind: https://local:54324
  cert-key:
  cert:
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

		nconf := conf.Network()
		t.Equal("https://local:54323", nconf.ConnInfo().URL().String())
		t.Equal("https://local:54324", nconf.Bind().String())

		t.NotNil(nconf.Certs())
		t.True(nconf.ConnInfo().Insecure())
	}

	{
		y := `
network:
  url: https://local:54323
  bind: https://local:54324
  cert-key: " "
  cert: " "
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

		nconf := conf.Network()
		t.Equal("https://local:54323", nconf.ConnInfo().URL().String())
		t.Equal("https://local:54324", nconf.Bind().String())

		t.NotNil(nconf.Certs())
		t.True(nconf.ConnInfo().Insecure())
	}
}

func (t *testConfigChecker) TestLocalNetworkWrongCertificates() {
	y := `
network:
  url: https://local:54323
  bind: https://local:54324
  cert-key: %s
  cert: %s
`

	// NOTE x509: certificate signed by unknown authority
	keyFile, certFile := t.newCertFiles("local")
	y = fmt.Sprintf(y, keyFile, certFile)

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

	nconf := conf.Network()
	t.Equal("https://local:54323", nconf.ConnInfo().URL().String())
	t.Equal("https://local:54324", nconf.Bind().String())

	t.NotNil(nconf.Certs())
	t.True(nconf.ConnInfo().Insecure())
}

func (t *testConfigChecker) TestLocalNetworkWrongHost() {
	y := `
network:
  url: https://local:54323
  bind: https://local:54324
  cert-key: %s
  cert: %s
`
	// NOTE certificate is valid for show.me, not local
	keyFile, certFile := t.newCertFiles("show.me")
	y = fmt.Sprintf(y, keyFile, certFile)

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

	nconf := conf.Network()
	t.Equal("https://local:54323", nconf.ConnInfo().URL().String())
	t.Equal("https://local:54324", nconf.Bind().String())

	t.NotNil(nconf.Certs())
	t.True(nconf.ConnInfo().Insecure())
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

func (t *testConfigChecker) TestEmptyLocalConfig() {
	y := ""
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

	t.NotNil(conf.LocalConfig())
	t.Equal(config.DefaultSyncInterval, conf.LocalConfig().SyncInterval())
}

func (t *testConfigChecker) TestLocalConfig() {
	{
		y := `
sync-interval: 3s
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

		t.NotNil(conf.LocalConfig())
		t.Equal(time.Second*3, conf.LocalConfig().SyncInterval())
	}

	{
		y := `
sync-interval: 3ms
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

		t.NotNil(conf.LocalConfig())
		t.Equal(time.Millisecond*3, conf.LocalConfig().SyncInterval())
	}
}

func TestConfigChecker(t *testing.T) {
	suite.Run(t, new(testConfigChecker))
}
