package process

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/launch/config"
	yamlconfig "github.com/spikeekips/mitum/launch/config/yaml"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"gopkg.in/yaml.v3"
)

const (
	ProcessNameConfig               = "config"
	HookNameConfigSuffrage          = "hook_process_suffrage"
	HookNameConfigProposalProcessor = "hook_proposal_processor"
	HookNameConfigGenesisOperations = "hook_genesis_operations"
	HookNameConfigVerbose           = "hook_config_verbose"
)

var ProcessorConfig pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameConfig,
		[]string{
			ProcessNameEncoders,
		},
		ProcessConfig,
	); err != nil {
		panic(err)
	} else {
		ProcessorConfig = i
	}
}

func ProcessConfig(ctx context.Context) (context.Context, error) {
	var source []byte
	if err := LoadConfigSourceContextValue(ctx, &source); err != nil {
		return ctx, err
	}

	var sourceType string
	if err := LoadConfigSourceTypeContextValue(ctx, &sourceType); err != nil {
		return ctx, err
	}

	if sourceType != "yaml" {
		return ctx, errors.Errorf("not supported config source type, %q", sourceType)
	}

	c, err := loadConfigYAML(ctx, source)
	if err != nil {
		return ctx, err
	}
	return checkConfig(c)
}

func loadConfigYAML(ctx context.Context, source []byte) (context.Context, error) {
	var yconf yamlconfig.LocalNode
	if err := yaml.Unmarshal(source, &yconf); err != nil {
		return ctx, err
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(source, &m); err != nil {
		return ctx, err
	}

	var enc *jsonenc.Encoder
	if err := config.LoadJSONEncoderContextValue(ctx, &enc); err != nil {
		return ctx, err
	}

	conf := config.NewBaseLocalNode(enc, m)
	ctx = context.WithValue(ctx, config.ContextValueConfig, conf)

	c, err := yconf.Set(ctx)
	if err != nil {
		return ctx, err
	}
	ctx = c

	return context.WithValue(ctx, config.ContextValueConfig, conf), nil
}

func checkConfig(ctx context.Context) (context.Context, error) {
	cc, err := config.NewChecker(ctx)
	if err != nil {
		return ctx, err
	}
	if err := util.NewChecker("config-checker", []util.CheckerFunc{
		cc.CheckLocalNetwork,
		cc.CheckStorage,
		cc.CheckPolicy,
	}).Check(); err != nil {
		if errors.Is(err, util.IgnoreError) {
			return ctx, nil
		}

		return ctx, err
	}
	return cc.Context(), nil
}

func Config(ps *pm.Processes) error {
	if err := ps.AddProcess(ProcessorConfig, false); err != nil {
		return err
	}

	if err := ps.AddHook(
		pm.HookPrefixPost, ProcessNameConfig,
		HookNameConfigSuffrage, HookSuffrageConfigFunc(DefaultHookHandlersSuffrageConfig),
		false,
	); err != nil {
		return err
	}

	if err := ps.AddHook(
		pm.HookPrefixPost, ProcessNameConfig,
		HookNameConfigProposalProcessor, HookProposalProcessorConfigFunc(DefaultHookHandlersProposalProcessorConfig),
		false,
	); err != nil {
		return err
	}

	if err := ps.AddHook(
		pm.HookPrefixPost, ProcessNameConfig,
		HookNameConfigGenesisOperations, HookGenesisOperationFunc(DefaultHookHandlersGenesisOperations),
		false,
	); err != nil {
		return err
	}

	if err := ps.AddHook(
		pm.HookPrefixPost, ProcessNameConfig,
		HookNameValidateConfig, HookValidateConfig,
		false,
	); err != nil {
		return err
	}

	return ps.AddHook(
		pm.HookPrefixPost, ProcessNameConfig,
		HookNameConfigVerbose, HookConfigVerbose,
		false,
	)
}
