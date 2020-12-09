package process

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base/operation"
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
		HookNameAddHinters, HookAddHinters(DefaultHinters),
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
address: node-010a:0.0.1
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
privatekey: KzmnCUoBrqYbkoP8AUki1AJsyKqxNsiqdrtTB2onyzQfB6MQ5Sef-0112:0.0.1
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
address: node-010a:0.0.1
nodes:
  - address: n0-010a:0.0.1
`
		ctx := t.loadConfig(y)

		va, err := config.NewValidator(ctx)
		t.NoError(err)
		_, err = va.CheckLocalNetwork()
		t.NoError(err)
		_, err = va.CheckNodes()
		t.Contains(err.Error(), "network of remote node is missing")
	}

	{
		y := `
address: node-010a:0.0.1
nodes:
  - address: n0-010a:0.0.1
    url: quic://local:54322
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
address: node-010a:0.0.1
nodes:
  - address: n0-010a:0.0.1
    url: quic://local:54322
    publickey: 27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb-0113:0.0.1
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
		t.Equal("n0-010a:0.0.1", conf.Nodes()[0].Address().String())
		t.Equal("quic://local:54322", conf.Nodes()[0].URL().String())
		t.Equal("27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb-0113:0.0.1", conf.Nodes()[0].Publickey().String())
	}
}

func (t *testConfigValidator) TestNodesSameAddressWithLocal() {
	y := `
address: n0-010a:0.0.1
nodes:
  - address: n0-010a:0.0.1
    url: quic://local:54322
    publickey: 27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb-0113:0.0.1
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
address: node-010a:0.0.1
nodes:
  - address: n0-010a:0.0.1
    url: quic://local:54322
    publickey: 27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb-0113:0.0.1
  - address: n0-010a:0.0.1
    url: quic://local:54323
    publickey: ideZAiLELe41jCqUD4zxmqqD7PXKR6uKS5MhZ8keqgcy-0113:0.0.1
`
	ctx := t.loadConfig(y)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckLocalNetwork()
	t.NoError(err)
	_, err = va.CheckNodes()
	t.Contains(err.Error(), "duplicated address found")
}

func (t *testConfigValidator) TestNodesSameNetworkWithLocal() {
	y := `
address: node-010a:0.0.1
network:
  url: quic://local:54322
nodes:
  - address: n0-010a:0.0.1
    url: quic://local:54322
    publickey: 27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb-0113:0.0.1
  - address: n1-010a:0.0.1
    url: quic://local:54323
    publickey: ideZAiLELe41jCqUD4zxmqqD7PXKR6uKS5MhZ8keqgcy-0113:0.0.1
`
	ctx := t.loadConfig(y)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckLocalNetwork()
	t.NoError(err)
	_, err = va.CheckNodes()
	t.Contains(err.Error(), "same network found with local")
}

func (t *testConfigValidator) TestNodesDuplicatedNetwork() {
	y := `
address: node-010a:0.0.1
nodes:
  - address: n0-010a:0.0.1
    url: quic://local:54322
    publickey: 27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb-0113:0.0.1
  - address: n1-010a:0.0.1
    url: quic://local:54322
    publickey: ideZAiLELe41jCqUD4zxmqqD7PXKR6uKS5MhZ8keqgcy-0113:0.0.1
`
	ctx := t.loadConfig(y)

	va, err := config.NewValidator(ctx)
	t.NoError(err)
	_, err = va.CheckLocalNetwork()
	t.NoError(err)
	_, err = va.CheckNodes()
	t.Contains(err.Error(), "duplicated network found")
}

func (t *testConfigValidator) TestEmptySuffrage() {
	y := `
suffrage:
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
}

func (t *testConfigValidator) TestFixedSuffrageWithEmptyAddress() {
	y := `
suffrage:
  type: fixed-suffrage
`
	ctx := t.loadConfig(y)

	_, err := HookSuffrageConfigFunc(DefaultHookHandlersSuffrageConfig)(ctx)
	t.Contains(err.Error(), "empty proposer and acting")
}

func (t *testConfigValidator) TestFixedSuffrageWithInvalidAddress() {
	y := `
suffrage:
  type: fixed-suffrage
  proposer: showme hahah
`
	ctx := t.loadConfig(y)

	_, err := HookSuffrageConfigFunc(DefaultHookHandlersSuffrageConfig)(ctx)
	t.Contains(err.Error(), "invalid proposer address for fixed-suffrage")
}

func (t *testConfigValidator) TestFixedSuffrage() {
	y := `
suffrage:
  type: fixed-suffrage
  proposer: n0-010a:0.0.1
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
	t.Equal("n0-010a:0.0.1", fs.Proposer.String())
	t.Empty(fs.Nodes)
}

func (t *testConfigValidator) TestFixedSuffrageWithNodes() {
	y := `
suffrage:
  type: fixed-suffrage
  proposer: n0-010a:0.0.1
  acting:
    - n1-010a:0.0.1
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
	t.Equal("n0-010a:0.0.1", fs.Proposer.String())
	t.Equal(1, len(fs.Nodes))
	t.Equal("n1-010a:0.0.1", fs.Nodes[0].String())
}

func (t *testConfigValidator) TestFixedSuffrageWithBadNodes() {
	y := `
suffrage:
  type: fixed-suffrage
  proposer: n0-010a:0.0.1
  acting:
    - n1-010a:0. # invalid address
`
	ctx := t.loadConfig(y)

	_, err := HookSuffrageConfigFunc(DefaultHookHandlersSuffrageConfig)(ctx)
	t.Contains(err.Error(), "invalid node address")
}

func (t *testConfigValidator) TestRoundrobin() {
	y := `
suffrage:
  type: roundrobin
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

	ctx, err := HookProposalProcessorFunc(DefaultHookHandlersProposalProcessor)(ctx)
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

	_, err := HookProposalProcessorFunc(DefaultHookHandlersProposalProcessor)(ctx)
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

func (t *testConfigValidator) TestLoadGenesisOperations() {
	y := `
privatekey: KzmnCUoBrqYbkoP8AUki1AJsyKqxNsiqdrtTB2onyzQfB6MQ5Sef-0112:0.0.1
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

func TestConfigValidator(t *testing.T) {
	suite.Run(t, new(testConfigValidator))
}
