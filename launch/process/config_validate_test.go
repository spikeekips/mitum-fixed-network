package process

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/launch"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
)

type testConfigValidator struct {
	suite.Suite
}

func (t *testConfigValidator) loadConfig(y string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextValueConfigSource, []byte(y))
	ctx = context.WithValue(ctx, ContextValueConfigSourceType, "yaml")

	ps := pm.NewProcesses().SetContext(ctx)
	t.NoError(ps.AddProcess(ProcessorEncoders, false))
	t.NoError(ps.AddProcess(ProcessorConfig, false))

	t.NoError(ps.AddHook(
		pm.HookPrefixPost, ProcessNameEncoders,
		HookNameAddHinters, HookAddHinters(launch.EncoderTypes, launch.EncoderHinters),
		true,
	))

	t.NoError(ps.Run())

	return ps.Context()
}

func (t *testConfigValidator) TestEmptyNodeAddress() {
	y := `
address:
`

	ctx := t.loadConfig(y)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckNodeAddress()
	t.Contains(err.Error(), "node address is missing")
}

func (t *testConfigValidator) TestMissingNodeAddress() {
	y := ""

	ctx := t.loadConfig(y)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckNodeAddress()
	t.Contains(err.Error(), "node address is missing")
}

func (t *testConfigValidator) TestNodeAddress() {
	y := `
address: node:sa-v0.0.1
`

	ctx := t.loadConfig(y)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckNodeAddress()
	t.NoError(err)
}

func (t *testConfigValidator) TestEmptyNodePrivatekey() {
	y := `
privatekey:
`

	ctx := t.loadConfig(y)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckNodePrivatekey()
	t.Contains(err.Error(), "node privatekey is missing")
}

func (t *testConfigValidator) TestMissingNodePrivatekey() {
	y := ""

	ctx := t.loadConfig(y)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckNodePrivatekey()
	t.Contains(err.Error(), "node privatekey is missing")
}

func (t *testConfigValidator) TestNodePrivatekey() {
	y := `
privatekey: KzmnCUoBrqYbkoP8AUki1AJsyKqxNsiqdrtTB2onyzQfB6MQ5Sef:btc-priv-v0.0.1
`

	ctx := t.loadConfig(y)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckNodePrivatekey()
	t.NoError(err)
}

func (t *testConfigValidator) TestNetworkID() {
	{
		y := `
network-id: show me
`

		ctx := t.loadConfig(y)

		va, err := config.NewValidator(ctx)
		t.NoError(err)
		_, err = va.CheckNetworkID()
		t.NoError(err)
	}
	{ // empty
		y := ""

		ctx := t.loadConfig(y)

		va, err := config.NewValidator(ctx)
		t.NoError(err)
		_, err = va.CheckNetworkID()
		t.Contains(err.Error(), "network id is missing")
	}
}

func (t *testConfigValidator) TestEmptyNodes() {
	{
		y := `
nodes:
`
		ctx := t.loadConfig(y)

		va, err := config.NewValidator(ctx)
		t.NoError(err)
		_, err = va.CheckStorage()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ctx, &conf))

		t.Nil(conf.Nodes())
	}
}

func (t *testConfigValidator) TestNodes() {
	{
		y := `
address: node:sa-v0.0.1
nodes:
  - address: n0:sa-v0.0.1
`
		ctx := t.loadConfig(y)

		va, err := config.NewValidator(ctx)
		t.NoError(err)
		_, err = va.CheckLocalNetwork()
		t.NoError(err)
		_, err = va.CheckNodes()
		t.Contains(err.Error(), "publickey of remote node is missing")
	}

	{
		y := `
address: node:sa-v0.0.1
nodes:
  - address: n0:sa-v0.0.1
`
		ctx := t.loadConfig(y)

		va, err := config.NewValidator(ctx)
		t.NoError(err)
		_, err = va.CheckLocalNetwork()
		t.NoError(err)
		_, err = va.CheckNodes()
		t.Contains(err.Error(), "publickey of remote node is missing")
	}

	{
		y := `
address: node:sa-v0.0.1
nodes:
  - address: n0:sa-v0.0.1
    publickey: 27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb:btc-pub-v0.0.1
`
		ctx := t.loadConfig(y)

		va, err := config.NewValidator(ctx)
		t.NoError(err)
		_, err = va.CheckLocalNetwork()
		t.NoError(err)
		_, err = va.CheckNodes()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ctx, &conf))

		t.Equal(1, len(conf.Nodes()))
		t.Equal("n0:sa-v0.0.1", conf.Nodes()[0].Address().String())
		t.Equal("27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb:btc-pub-v0.0.1", conf.Nodes()[0].Publickey().String())
		t.Nil(conf.Nodes()[0].ConnInfo())
	}
}

func (t *testConfigValidator) TestNodesWithConnInfo() {
	y := `
address: node:sa-v0.0.1
nodes:
  - address: n0:sa-v0.0.1
    publickey: 27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb:btc-pub-v0.0.1
    url: https://findme/showme?findme=true
`
	ctx := t.loadConfig(y)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckLocalNetwork()
	t.NoError(err)
	_, err = va.CheckNodes()
	t.NoError(err)

	var conf config.LocalNode
	t.NoError(config.LoadConfigContextValue(ctx, &conf))

	t.Equal(1, len(conf.Nodes()))
	t.Equal("n0:sa-v0.0.1", conf.Nodes()[0].Address().String())
	t.Equal("27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb:btc-pub-v0.0.1", conf.Nodes()[0].Publickey().String())
	t.Equal("https://findme:443/showme?findme=true", conf.Nodes()[0].ConnInfo().URL().String())
	t.False(conf.Nodes()[0].ConnInfo().Insecure())
}

func (t *testConfigValidator) TestNodesWithConnInfoTLSInsecure() {
	y := `
address: node:sa-v0.0.1
nodes:
  - address: n0:sa-v0.0.1
    publickey: 27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb:btc-pub-v0.0.1
    url: https://findme/showme?findme=true
    tls-insecure: true
`
	ctx := t.loadConfig(y)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckLocalNetwork()
	t.NoError(err)
	_, err = va.CheckNodes()
	t.NoError(err)

	var conf config.LocalNode
	t.NoError(config.LoadConfigContextValue(ctx, &conf))

	t.Equal(1, len(conf.Nodes()))
	t.Equal("n0:sa-v0.0.1", conf.Nodes()[0].Address().String())
	t.Equal("27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb:btc-pub-v0.0.1", conf.Nodes()[0].Publickey().String())
	t.Equal("https://findme:443/showme?findme=true", conf.Nodes()[0].ConnInfo().URL().String())
	t.True(conf.Nodes()[0].ConnInfo().Insecure())
}

func (t *testConfigValidator) TestNodesSameAddressWithLocal() {
	y := `
address: n0:sa-v0.0.1
nodes:
  - address: n0:sa-v0.0.1
    publickey: 27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb:btc-pub-v0.0.1
`
	ctx := t.loadConfig(y)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckLocalNetwork()
	t.NoError(err)
	_, err = va.CheckNodes()
	t.Contains(err.Error(), "same address found with local node")
}

func (t *testConfigValidator) TestNodesDuplicatedAddress() {
	y := `
address: node:sa-v0.0.1
nodes:
  - address: n0:sa-v0.0.1
    publickey: 27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb:btc-pub-v0.0.1
  - address: n0:sa-v0.0.1
    publickey: ideZAiLELe41jCqUD4zxmqqD7PXKR6uKS5MhZ8keqgcy:btc-pub-v0.0.1
`
	ctx := t.loadConfig(y)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckLocalNetwork()
	t.NoError(err)
	_, err = va.CheckNodes()
	t.Contains(err.Error(), "duplicated address found")
}

func (t *testConfigValidator) TestEmptySuffrage() {
	y := `
address: n0:sa-v0.0.1
`
	ctx := t.loadConfig(y)

	_, err := HookSuffrageConfigFunc(nil)(ctx)
	t.NoError(err)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckNodes()
	t.NoError(err)

	_, err = va.CheckSuffrage()
	t.Contains(err.Error(), "suffrage nodes and nodes both empty")
}

func (t *testConfigValidator) TestEmptySuffrageWithoutNodes() {
	y := `
address: n0:sa-v0.0.1
suffrage:
`
	ctx := t.loadConfig(y)

	_, err := HookSuffrageConfigFunc(nil)(ctx)
	t.NoError(err)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckNodes()
	t.NoError(err)

	_, err = va.CheckSuffrage()
	t.Contains(err.Error(), "suffrage nodes and nodes both empty")
}

func (t *testConfigValidator) TestEmptySuffrageWithWrongNode() {
	y := `
suffrage:
  nodes:
    - n0
`
	ctx := t.loadConfig(y)

	_, err := HookSuffrageConfigFunc(nil)(ctx)
	t.Contains(err.Error(), "invalid node address")
}

func (t *testConfigValidator) TestEmptySuffrageWithNodes() {
	y := `
suffrage:
  nodes:
`
	ctx := t.loadConfig(y)

	_, err := HookSuffrageConfigFunc(nil)(ctx)
	t.Contains(err.Error(), "invalid nodes list")
}

func (t *testConfigValidator) TestSuffrageUnknownNode() {
	y := `
address: n0:sa-v0.0.1

nodes:
  - address: n1:sa-v0.0.1
    publickey: 27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb:btc-pub-v0.0.1

suffrage:
  nodes:
    - unknown:sa-v0.0.1
`
	ctx := t.loadConfig(y)

	ctx, err := HookSuffrageConfigFunc(nil)(ctx)
	t.NoError(err)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckNodes()
	t.NoError(err)

	_, err = va.CheckSuffrage()
	t.Contains(err.Error(), " in suffrage not found in nodes")
}

func (t *testConfigValidator) TestSuffrage() {
	y := `
address: n0:sa-v0.0.1

nodes:
  - address: n1:sa-v0.0.1
    publickey: 27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb:btc-pub-v0.0.1

suffrage:
  nodes:
    - n0:sa-v0.0.1
`
	ctx := t.loadConfig(y)

	ctx, err := HookSuffrageConfigFunc(nil)(ctx)
	t.NoError(err)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckSuffrage()
	t.NoError(err)

	var conf config.LocalNode
	t.NoError(config.LoadConfigContextValue(ctx, &conf))

	t.IsType(config.RoundrobinSuffrage{}, conf.Suffrage()) // NOTE empty will return default suffrage.

	t.Equal(1, len(conf.Suffrage().Nodes()))
}

func (t *testConfigValidator) TestFixedSuffrageWithEmptyAddress() {
	y := `
suffrage:
  type: fixed-suffrage
  nodes:
    - n0:sa-v0.0.1
`
	ctx := t.loadConfig(y)

	_, err := HookSuffrageConfigFunc(DefaultHookHandlersSuffrageConfig)(ctx)
	t.Contains(err.Error(), "empty proposer")
}

func (t *testConfigValidator) TestFixedSuffrageWithInvalidAddress() {
	y := `
suffrage:
  type: fixed-suffrage
  proposer: showme hahah
  nodes:
    - n0:sa-v0.0.1
`
	ctx := t.loadConfig(y)

	_, err := HookSuffrageConfigFunc(DefaultHookHandlersSuffrageConfig)(ctx)
	t.Contains(err.Error(), "invalid proposer address for fixed-suffrage")
}

func (t *testConfigValidator) TestFixedSuffrage() {
	y := `
address: n0:sa-v0.0.1

nodes:
  - address: n1:sa-v0.0.1
    publickey: 27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb:btc-pub-v0.0.1

suffrage:
  type: fixed-suffrage
  proposer: n0:sa-v0.0.1
  nodes:
    - n0:sa-v0.0.1
`
	ctx := t.loadConfig(y)

	ctx, err := HookSuffrageConfigFunc(DefaultHookHandlersSuffrageConfig)(ctx)
	t.NoError(err)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckSuffrage()
	t.NoError(err)

	var conf config.LocalNode
	t.NoError(config.LoadConfigContextValue(ctx, &conf))

	t.IsType(config.FixedSuffrage{}, conf.Suffrage())

	fs := conf.Suffrage().(config.FixedSuffrage)
	t.Equal("n0:sa-v0.0.1", fs.Proposer.String())
	t.NotEmpty(fs.Nodes())
}

func (t *testConfigValidator) TestFixedSuffrageWithNodes() {
	y := `
address: n0:sa-v0.0.1

nodes:
  - address: n1:sa-v0.0.1
    publickey: 27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb:btc-pub-v0.0.1

suffrage:
  type: fixed-suffrage
  proposer: n0:sa-v0.0.1
  nodes:
    - n1:sa-v0.0.1
`
	ctx := t.loadConfig(y)

	ctx, err := HookSuffrageConfigFunc(DefaultHookHandlersSuffrageConfig)(ctx)
	t.NoError(err)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckSuffrage()
	t.NoError(err)

	var conf config.LocalNode
	t.NoError(config.LoadConfigContextValue(ctx, &conf))

	t.IsType(config.FixedSuffrage{}, conf.Suffrage())

	fs := conf.Suffrage().(config.FixedSuffrage)
	t.Equal("n0:sa-v0.0.1", fs.Proposer.String())
	t.Equal(1, len(fs.Nodes()))
	t.Equal("n1:sa-v0.0.1", fs.Nodes()[0].String())
}

func (t *testConfigValidator) TestFixedSuffrageWithBadNodes() {
	y := `
suffrage:
  type: fixed-suffrage
  proposer: n0:sa-v0.0.1
  nodes:
    - n1-010a:0. # invalid address
`
	ctx := t.loadConfig(y)

	_, err := HookSuffrageConfigFunc(DefaultHookHandlersSuffrageConfig)(ctx)
	t.Contains(err.Error(), "invalid node address")
}

func (t *testConfigValidator) TestRoundrobin() {
	y := `
address: n0:sa-v0.0.1

nodes:
  - address: n1:sa-v0.0.1
    publickey: 27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb:btc-pub-v0.0.1

suffrage:
  type: roundrobin
  nodes:
    - n1:sa-v0.0.1
`
	ctx := t.loadConfig(y)

	ctx, err := HookSuffrageConfigFunc(DefaultHookHandlersSuffrageConfig)(ctx)
	t.NoError(err)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckSuffrage()
	t.NoError(err)

	var conf config.LocalNode
	t.NoError(config.LoadConfigContextValue(ctx, &conf))

	t.IsType(config.RoundrobinSuffrage{}, conf.Suffrage())
}

func (t *testConfigValidator) TestEmptyProposalProcessor() {
	y := `
proposal-processor:
`
	ctx := t.loadConfig(y)

	ctx, err := HookProposalProcessorConfigFunc(DefaultHookHandlersProposalProcessorConfig)(ctx)
	t.NoError(err)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckProposalProcessor()
	t.NoError(err)

	var conf config.LocalNode
	t.NoError(config.LoadConfigContextValue(ctx, &conf))

	t.IsType(config.DefaultProposalProcessor{}, conf.ProposalProcessor())
}

func (t *testConfigValidator) TestUnknownProposalProcessor() {
	y := `
proposal-processor:
  type: find-me
`
	ctx := t.loadConfig(y)

	_, err := HookProposalProcessorConfigFunc(DefaultHookHandlersProposalProcessorConfig)(ctx)
	t.Contains(err.Error(), "unknown proposal-processor found, find-me")
}

func testGenesisOperationsHandlerSetPolicy(ctx context.Context, m map[string]interface{}) (operation.Operation, error) {
	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return nil, err
	}

	var key string
	var value []byte
	for k := range m {
		if k == "type" {
			continue
		}

		key = k
		value = []byte(m[k].(string))
	}

	return operation.NewKVOperation(conf.Privatekey(), DefaultGenesisOperationToken, key, value, conf.NetworkID())
}

func (t *testConfigValidator) TestErrorProposalProcessor() {
	y := `
proposal-processor:
   type: error
   when-prepare:
       - point: 3,1
       - point: 4,2
         type: wrong-block
   when-save:
       - point: 5,3
       - point: 6,4
`
	ctx := t.loadConfig(y)

	ctx, err := HookProposalProcessorConfigFunc(DefaultHookHandlersProposalProcessorConfig)(ctx)
	t.NoError(err)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckProposalProcessor()
	t.NoError(err)

	var conf config.LocalNode
	t.NoError(config.LoadConfigContextValue(ctx, &conf))

	t.IsType(config.ErrorProposalProcessor{}, conf.ProposalProcessor())

	c := conf.ProposalProcessor().(config.ErrorProposalProcessor)
	t.Equal([]config.ErrorPoint{{Height: 3, Round: 1, Type: config.ErrorTypeError}, {Height: 4, Round: 2, Type: config.ErrorTypeWrongBlockHash}}, c.WhenPreparePoints)
	t.Equal([]config.ErrorPoint{{Height: 5, Round: 3, Type: config.ErrorTypeError}, {Height: 6, Round: 4, Type: config.ErrorTypeError}}, c.WhenSavePoints)
}

func (t *testConfigValidator) TestLoadGenesisOperations() {
	y := `
privatekey: KzmnCUoBrqYbkoP8AUki1AJsyKqxNsiqdrtTB2onyzQfB6MQ5Sef:btc-priv-v0.0.1
network-id: show me
genesis-operations:
  - type: set-data
    suffrage-nodes: "empty"
`
	ctx := t.loadConfig(y)

	handlers := map[string]HookHandlerGenesisOperations{}
	for k, v := range DefaultHookHandlersGenesisOperations {
		handlers[k] = v
	}
	handlers["set-data"] = testGenesisOperationsHandlerSetPolicy
	ctx, err := HookGenesisOperationFunc(handlers)(ctx)
	t.NoError(err)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckGenesisOperations()
	t.NoError(err)

	var conf config.LocalNode
	t.NoError(config.LoadConfigContextValue(ctx, &conf))

	t.Equal(1, len(conf.GenesisOperations()))

	op := conf.GenesisOperations()[0]
	t.NoError(op.IsValid(conf.NetworkID()))

	t.IsType(operation.KVOperation{}, op)
	t.Implements((*operation.OperationFact)(nil), op.Fact())

	kop := op.(operation.KVOperation)

	t.Equal(DefaultGenesisOperationToken, kop.Fact().(operation.KVOperationFact).Token())
	t.Equal("suffrage-nodes", kop.Key())
	t.Equal([]byte("empty"), kop.Value())
}

func (t *testConfigValidator) TestUnknownGenesisOperations() {
	y := `
genesis-operations:
  - type: kill-me
`
	ctx := t.loadConfig(y)

	_, err := HookGenesisOperationFunc(DefaultHookHandlersGenesisOperations)(ctx)
	t.Contains(err.Error(), "invalid genesis operation found")
}

func (t *testConfigValidator) TestLocalConfigEmptySyncInterval() {
	{
		y := ""

		ctx := t.loadConfig(y)

		va, err := config.NewValidator(ctx)
		t.NoError(err)
		_, err = va.CheckLocalConfig()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ctx, &conf))

		t.Equal(config.DefaultSyncInterval, conf.LocalConfig().SyncInterval())
	}

	{
		y := `
sync-interval:
`

		ctx := t.loadConfig(y)

		va, err := config.NewValidator(ctx)
		t.NoError(err)
		_, err = va.CheckLocalConfig()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ctx, &conf))

		t.Equal(config.DefaultSyncInterval, conf.LocalConfig().SyncInterval())
	}
}

func (t *testConfigValidator) TestLocalConfigTooNarrowSyncInterval() {
	y := `
sync-interval: 900ms
`

	ctx := t.loadConfig(y)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckLocalConfig()
	t.Contains(err.Error(), "sync-interval too narrow")
}

func (t *testConfigValidator) TestLocalConfigSyncInterval() {
	y := `
sync-interval: 9s
`

	ctx := t.loadConfig(y)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckLocalConfig()
	t.NoError(err)

	var conf config.LocalNode
	t.NoError(config.LoadConfigContextValue(ctx, &conf))

	t.Equal(time.Second*9, conf.LocalConfig().SyncInterval())
}

func (t *testConfigValidator) TestLocalConfigEmptTimeServer() {
	{
		y := ""

		ctx := t.loadConfig(y)

		va, err := config.NewValidator(ctx)
		t.NoError(err)
		_, err = va.CheckLocalConfig()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ctx, &conf))

		t.Equal(config.DefaultTimeServer, conf.LocalConfig().TimeServer())
	}

	{
		y := `
time-server:
`

		ctx := t.loadConfig(y)

		va, err := config.NewValidator(ctx)
		t.NoError(err)
		_, err = va.CheckLocalConfig()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ctx, &conf))

		t.Equal(config.DefaultTimeServer, conf.LocalConfig().TimeServer())
	}

	{
		y := `
time-server: ""
`

		ctx := t.loadConfig(y)

		va, err := config.NewValidator(ctx)
		t.NoError(err)
		_, err = va.CheckLocalConfig()
		t.NoError(err)

		var conf config.LocalNode
		t.NoError(config.LoadConfigContextValue(ctx, &conf))

		t.Empty(conf.LocalConfig().TimeServer())
	}
}

func (t *testConfigValidator) TestLocalConfigTimeServer() {
	y := `
time-server: time.kriss.re.kr
`

	ctx := t.loadConfig(y)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckLocalConfig()
	t.NoError(err)

	var conf config.LocalNode
	t.NoError(config.LoadConfigContextValue(ctx, &conf))

	t.Equal("time.kriss.re.kr", conf.LocalConfig().TimeServer())
}

func TestConfigValidator(t *testing.T) {
	suite.Run(t, new(testConfigValidator))
}
