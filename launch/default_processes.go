package launch

import (
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
)

var defaultProcesses = []pm.Process{
	process.ProcessorEncoders,
	process.ProcessorBlockFS,
	process.ProcessorStorage,
	process.ProcessorLocal,
	process.ProcessorProposalProcessor,
	process.ProcessorSuffrage,
	process.ProcessorConsensusStates,
	process.ProcessorNetwork,
	process.ProcessorStartNetwork,
	process.ProcessorStartConsensusStates,
}

var defaultHooks = []pm.Hook{
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameEncoders,
		process.HookNameAddHinters, process.HookAddHinters(process.DefaultHinters)),
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameNetwork,
		process.HookNameSetNetworkHandlers, process.HookSetNetworkHandlers),
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameLocal, process.HookNameSetPolicy, process.HookSetPolicy),
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameLocal, process.HookNameRemoteNodes, process.HookRemoteNodes),
}

func DefaultProcesses() *pm.Processes {
	ps := pm.NewProcesses()

	if err := process.Config(ps); err != nil {
		panic(err)
	}

	for i := range defaultProcesses {
		if err := ps.AddProcess(defaultProcesses[i], false); err != nil {
			panic(err)
		}
	}

	for i := range defaultHooks {
		hook := defaultHooks[i]
		if err := ps.AddHook(hook.Prefix, hook.Process, hook.Name, hook.F, true); err != nil {
			panic(err)
		}
	}

	return ps
}
