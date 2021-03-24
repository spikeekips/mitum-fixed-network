package process

import (
	"context"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

const HookNameCheckEmptyBlock = "check_empty_block"

// HookCheckEmptyBlock checks whether local has empty block. If empty block and
// there are no other nodes, stop process with error.
func HookCheckEmptyBlock(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var policy *isaac.LocalPolicy
	if err := LoadPolicyContextValue(ctx, &policy); err != nil {
		return ctx, err
	}

	var suffrage base.Suffrage
	if err := LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return ctx, err
	}

	var st storage.Database
	if err := LoadDatabaseContextValue(ctx, &st); err != nil {
		return ctx, err
	}

	var blockData blockdata.BlockData
	if err := LoadBlockDataContextValue(ctx, &blockData); err != nil {
		return ctx, err
	}

	if m, err := storage.CheckBlockEmpty(st); err != nil {
		return ctx, err
	} else if m == nil {
		log.Debug().Msg("empty block found; storage will be empty")

		if err := blockdata.Clean(st, blockData, false); err != nil {
			return nil, err
		}

		if len(suffrage.Nodes()) < 2 {
			return ctx, xerrors.Errorf("empty block, but no other nodes; can not sync")
		}
	} else if err := m.IsValid(policy.NetworkID()); err != nil {
		return ctx, xerrors.Errorf("invalid block found, clean up block: %w", err)
	} else {
		log.Debug().Hinted("block", m).Msg("valid initial block found")
	}

	return ctx, nil
}
