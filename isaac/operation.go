package isaac

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

var KnownSealError = util.NewError("seal is known")

type OperationsExtractor struct {
	*logging.Logging
	local    base.Address
	proposer base.Address
	database storage.Database
	nodepool *network.Nodepool
	opsh     []valuehash.Hash
	founds   map[string]struct{}
}

func NewOperationsExtractor(
	local base.Address,
	proposer base.Address,
	db storage.Database,
	nodepool *network.Nodepool,
	opsh []valuehash.Hash,
) *OperationsExtractor {
	return &OperationsExtractor{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "seals-extracter").
				Stringer("proposer", proposer).
				Int("ops", len(opsh))
		}),
		local:    local,
		database: db,
		nodepool: nodepool,
		proposer: proposer,
		opsh:     opsh,
		founds:   map[string]struct{}{},
	}
}

func (se *OperationsExtractor) Extract(ctx context.Context) ([]operation.Operation, error) {
	se.Log().Debug().Msg("trying to extract seals")

	l, notFounds, err := se.extractFromStorage(ctx)
	if err != nil {
		return nil, err
	}

	if len(notFounds) > 0 {
		i, err := se.extractFromChannel(ctx, notFounds)
		if err != nil {
			return nil, err
		}

		for k := range i {
			l[k] = i[k]
		}
	}

	if len(l) < len(se.opsh) {
		return nil, errors.Errorf("some operations are missing, %d", len(se.opsh)-len(l))
	}

	se.Log().Debug().Int("operations", len(l)).Msg("extracted operations")

	ops := make([]operation.Operation, len(se.opsh))

	for i := range se.opsh {
		h := se.opsh[i]
		j, found := l[h.String()]
		if !found {
			return nil, errors.Errorf("some operation is missing, %v", h)
		}
		ops[i] = j
	}

	return ops, nil
}

func (se *OperationsExtractor) extractFromStorage(_ context.Context) (
	map[string]operation.Operation,
	[]valuehash.Hash,
	error,
) {
	ops, err := se.database.StagedOperationsByFact(se.opsh)
	if err != nil {
		return nil, nil, err
	}

	se.Log().Debug().Int("operations", len(ops)).Int("not_founds", len(se.opsh)-len(ops)).Msg("extracted from storage")

	extracted := map[string]operation.Operation{}
	for i := range ops {
		op := ops[i]
		extracted[op.Fact().Hash().String()] = op
	}

	var notFounds []valuehash.Hash
	if len(extracted) != len(se.opsh) {
		for i := range se.opsh {
			h := se.opsh[i]
			if _, found := extracted[h.String()]; found {
				continue
			}
			notFounds = append(notFounds, h)
		}
	}

	return extracted, notFounds, nil
}

func (se *OperationsExtractor) extractFromChannel(
	ctx context.Context, notFounds []valuehash.Hash,
) (map[string]operation.Operation, error) {
	if se.local.Equal(se.proposer) {
		return nil, errors.Errorf("proposer is local, but it does not have operations. Hmmm")
	}

	_, proposerch, found := se.nodepool.Node(se.proposer)
	if !found {
		return nil, errors.Errorf("proposer is not in nodes: %v", se.proposer)
	} else if proposerch == nil {
		return nil, errors.Errorf("proposer is dead: %v", se.proposer)
	}

	received, err := proposerch.StagedOperations(ctx, notFounds)
	if err != nil {
		return nil, err
	}

	if err := se.database.NewOperations(received); err != nil {
		if !errors.Is(err, util.DuplicatedError) {
			return nil, err
		}
	}

	ops := map[string]operation.Operation{}
	for i := range received {
		op := received[i]
		ops[op.Fact().Hash().String()] = op
	}

	return ops, nil
}
