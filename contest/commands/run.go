package commands

import (
	"sync"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/contest/common"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type RunCommand struct {
	Nodes     uint   `args:"" default:"${nodes}" help:"number of suffrage nodes"`
	NetworkID string `args:"" default:"${networkID}" help:"network id"`
	// TODO select network type
}

func (cm RunCommand) generateBlocks(ns []*isaac.Localstate) error {
	if bg, err := isaac.NewDummyBlocksV0Generator(
		ns[0],
		3,
		common.NewSuffrage(ns[0]),
		ns,
	); err != nil {
		return xerrors.Errorf("failed new DummyBlocksV0Generator: %w", err)
	} else if err := bg.Generate(true); err != nil {
		return xerrors.Errorf("failed to generate initial blocks: %w", err)
	}

	return nil
}

func (cm RunCommand) createNodeProcess(
	localstate *isaac.Localstate,
	log logging.Logger,
) (*common.NodeProcess, error) {
	np, err := common.NewNodeProcess(localstate)
	if err != nil {
		return nil, err
	}

	l := log.WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("node", np.Localstate.Node().Address())
	})

	_ = np.SetLogger(l) // TODO set verbose

	{
		b, err := util.JSONMarshal(np.Localstate)
		if err != nil {
			return nil, err
		}
		np.Log().Debug().RawJSON("local_states", b).Msg("node process created")
	}

	return np, nil
}

func (cm RunCommand) startNodes(nodeProcesses []*common.NodeProcess, exitHooks *[]func(), log logging.Logger) error {
	var wg sync.WaitGroup
	wg.Add(len(nodeProcesses))

	errChan := make(chan error)
	for _, np := range nodeProcesses {
		np := np

		*exitHooks = append(*exitHooks, func() {
			_ = np.Stop()
		})

		go func(np *common.NodeProcess) {
			errChan <- np.Start()
			wg.Done()
		}(np)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err == nil {
			continue
		}

		log.Error().Err(err).Msg("failed to start NodeProcess")

		return err
	}

	return nil
}

func (cm RunCommand) Run(_ *CommonFlags, log logging.Logger, exitHooks *[]func()) error {
	var ns []*isaac.Localstate
	for i := 0; i < int(cm.Nodes); i++ {
		if nl, err := common.NewNode(i, []byte(cm.NetworkID), "quic"); err != nil {
			return err
		} else {
			ns = append(ns, nl)
		}
	}

	for _, nl := range ns {
		for _, other := range ns {
			if nl.Node().Address().Equal(other.Node().Address()) {
				continue
			}
			if err := nl.Nodes().Add(other.Node()); err != nil {
				return err
			}
		}

		threshold, err := base.NewThreshold(uint(nl.Nodes().Len()+1), 67)
		if err != nil {
			return err
		}
		_ = nl.Policy().SetThreshold(threshold)
	}

	if err := cm.generateBlocks(ns); err != nil {
		return err
	}

	nps := make([]*common.NodeProcess, len(ns))
	for i, nl := range ns {
		np, err := cm.createNodeProcess(nl, log)
		if err != nil {
			return xerrors.Errorf("failed to create NodeProcess: %w", err)
		}
		nps[i] = np
	}

	for _, np := range nps {
		var nodes []*isaac.Localstate
		for _, other := range nps {
			nodes = append(nodes, other.Localstate)
		}

		np.AllNodes = nodes
	}

	if err := cm.startNodes(nps, exitHooks, log); err != nil {
		return err
	}

	return common.LongRunningCommandError
}
