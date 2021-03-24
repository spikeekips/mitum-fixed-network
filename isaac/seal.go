package isaac

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/errors"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

var KnownSealError = errors.NewError("seal is known")

type SealsExtracter struct {
	*logging.Logging
	local    base.Address
	proposer base.Address
	st       storage.Database
	nodepool *network.Nodepool
	seals    []valuehash.Hash
	founds   map[string]struct{}
}

func NewSealsExtracter(
	local base.Address,
	proposer base.Address,
	st storage.Database,
	nodepool *network.Nodepool,
	seals []valuehash.Hash,
) *SealsExtracter {
	return &SealsExtracter{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "seals-extracter").
				Hinted("proposer", proposer).
				Int("seals", len(seals))
		}),
		local:    local,
		st:       st,
		nodepool: nodepool,
		proposer: proposer,
		seals:    seals,
		founds:   map[string]struct{}{},
	}
}

func (se *SealsExtracter) Extract(ctx context.Context) ([]operation.Operation, error) {
	se.Log().Debug().Msg("trying to extract seals")

	var notFounds []valuehash.Hash
	var opsCount int
	opsBySeals := map[string][]operation.Operation{}

	if i, f, err := se.extractFromStorage(ctx, opsBySeals); err != nil {
		return nil, err
	} else {
		opsCount += i
		notFounds = f
	}

	if len(notFounds) > 0 {
		if i, err := se.extractFromChannel(ctx, notFounds, opsBySeals); err != nil {
			return nil, err
		} else {
			opsCount += i
		}
	}

	se.Log().Debug().Int("operations", opsCount).Msg("extracted seals and it's operations")

	ops := make([]operation.Operation, opsCount)

	var offset int
	for i := range se.seals {
		h := se.seals[i]
		if l, found := opsBySeals[h.String()]; !found {
			continue
		} else if len(l) > 0 {
			copy(ops[offset:], l)
			offset += len(l)
		}
	}

	return ops, nil
}

func (se *SealsExtracter) extractFromStorage(
	ctx context.Context,
	opsBySeals map[string][]operation.Operation,
) (int, []valuehash.Hash, error) {
	var notFounds []valuehash.Hash
	var count int
	for i := range se.seals {
		h := se.seals[i]
		switch ops, found, err := se.fromStorage(ctx, h); {
		case err != nil:
			if xerrors.Is(err, context.DeadlineExceeded) || xerrors.Is(err, context.Canceled) {
				return count, nil, err
			} else if xerrors.Is(err, storage.NotFoundError) {
				notFounds = append(notFounds, h)

				continue
			}

			return count, nil, err
		case !found:
			notFounds = append(notFounds, h)

			continue
		default:
			opsBySeals[h.String()] = se.filterFounds(ops)
			count += len(opsBySeals[h.String()])
		}
	}

	se.Log().Debug().Int("operations", count).Msg("extracted from storage")

	return count, notFounds, nil
}

func (se *SealsExtracter) extractFromChannel(
	ctx context.Context,
	seals []valuehash.Hash,
	opsBySeals map[string][]operation.Operation,
) (int, error) {
	finished := make(chan error)

	var m map[string][]operation.Operation
	go func() {
		i, err := se.fromChannel(seals)
		if err == nil {
			m = i
		}

		finished <- err
	}()

	var count int
	select {
	case <-ctx.Done():
		return count, ctx.Err()
	case err := <-finished:
		if err != nil {
			return count, err
		}
	}

	for k := range m {
		opsBySeals[k] = se.filterFounds(m[k])
		count += len(opsBySeals[k])
	}

	se.Log().Debug().Int("operations", count).Msg("extracted from remote")

	return count, nil
}

func (se *SealsExtracter) filterFounds(ops []operation.Operation) []operation.Operation {
	if len(ops) < 1 {
		return nil
	}

	var nops []operation.Operation
	for i := range ops {
		h := ops[i].Hash().String()
		if _, found := se.founds[h]; found {
			continue
		} else {
			nops = append(nops, ops[i])

			se.founds[h] = struct{}{}
		}
	}

	return nops
}

func (se *SealsExtracter) filterDuplicated(ops []operation.Operation) []operation.Operation {
	if len(ops) < 1 {
		return nil
	}

	founds := map[string]struct{}{}
	var nops []operation.Operation
	for i := range ops {
		op := ops[i]
		fk := op.Hash().String()
		if _, found := founds[fk]; found {
			continue
		} else {
			nops = append(nops, op)
			founds[fk] = struct{}{}
		}
	}

	return nops
}

func (se *SealsExtracter) fromStorage(
	ctx context.Context,
	h valuehash.Hash, /* seal.Hash() */
) ([]operation.Operation, bool, error) {
	var ops []operation.Operation
	var found bool
	f := func(h valuehash.Hash) error {
		if sl, found0, err := se.st.Seal(h); err != nil {
			return err
		} else if !found0 {
			return nil
		} else if os, ok := sl.(operation.Seal); !ok {
			return xerrors.Errorf("not operation.Seal: %T", sl)
		} else {
			ops = se.filterDuplicated(os.Operations())
			found = true

			return nil
		}
	}

	finished := make(chan error)
	go func() {
		finished <- f(h)
	}()

	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	case err := <-finished:
		if err != nil {
			return nil, false, err
		}
	}

	return ops, found, nil
}

func (se *SealsExtracter) fromChannel(notFounds []valuehash.Hash) (map[string][]operation.Operation, error) {
	var proposer network.Node
	if se.local.Equal(se.proposer) {
		return nil, xerrors.Errorf("proposer is local, but it does not have seals. Hmmm")
	} else if node, found := se.nodepool.Node(se.proposer); !found {
		return nil, xerrors.Errorf("proposer is not in nodes: %v", se.proposer)
	} else {
		proposer = node
	}

	received, err := proposer.Channel().Seals(context.TODO(), notFounds)
	if err != nil {
		return nil, err
	}

	if err := se.st.NewSeals(received); err != nil {
		if !xerrors.Is(err, storage.DuplicatedError) {
			return nil, err
		}
	}

	bySeals := map[string][]operation.Operation{}
	for i := range received {
		sl := received[i]
		if os, ok := sl.(operation.Seal); !ok {
			return nil, xerrors.Errorf("not operation.Seal: %T", sl)
		} else {
			bySeals[sl.Hash().String()] = os.Operations()
		}
	}

	return bySeals, nil
}
