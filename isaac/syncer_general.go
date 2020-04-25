package isaac

import (
	"sort"
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

/*
GeneralSyncer will sync for the closed network.

> The closed network means that the network does not allowed the anonymous node
to enter the network.

GeneralSyncer has the managed suffrage members, so there are the specific
sources to fetch the blocks.

Before starting GeneralSyncer, these sources should be specified.

1. GeneralSyncer will try to fetch the manifest from all of them.
1. and then will compare the fetched manifests.
1. if some nodes does not respond, that node will be ignored.
1. the fetched data from nodes should be over threshold(2/3).

> 'from' and 'to' is not index number. If from=1 and to=5, GeneralSyncer
will sync [1,2,3,4,5].
*/
type GeneralSyncer struct { // nolint; maligned
	sync.RWMutex
	*logging.Logging
	localstate              *Localstate
	storage                 storage.SyncerStorage
	sourceNodes             map[base.Address]Node
	heightFrom              base.Height
	heightTo                base.Height
	limitManifestsPerWorker int
	limitBlocksPerOnce      int
	pn                      []base.Address
	st                      SyncerState
	baseManifest            block.Manifest
	willSave                bool
	stateChan               chan<- Syncer
}

func NewGeneralSyncer(
	localstate *Localstate,
	sourceNodes []Node,
	from, to base.Height,
) (*GeneralSyncer, error) {
	switch {
	case from > to:
		return nil, xerrors.Errorf("from height is same or higher than to height")
	case len(sourceNodes) < 1:
		return nil, xerrors.Errorf("empty source nodes")
	}

	if lastBlock := localstate.LastBlock(); lastBlock != nil {
		if from <= lastBlock.Height() {
			return nil, xerrors.Errorf("from height is same or lower than last block; from=%d last=%d", from, lastBlock.Height())
		}
	}

	sn := map[base.Address]Node{}
	{
		for _, node := range sourceNodes {
			if localstate.Node().Address().Equal(node.Address()) {
				return nil, xerrors.Errorf("one of sourceNodes is same with local node")
			}

			if _, found := sn[node.Address()]; found {
				return nil, xerrors.Errorf("duplicated node found")
			}

			sn[node.Address()] = node
		}
	}

	cs := &GeneralSyncer{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.
				Hinted("from", from).
				Hinted("to", to).
				Str("module", "general-syncer")
		}),
		localstate: localstate,
		//storage:                 localstate.Storage().SyncerStorage(),
		sourceNodes:             sn,
		heightFrom:              from,
		heightTo:                to,
		limitManifestsPerWorker: 100, // set default
		limitBlocksPerOnce:      100, // set default
		st:                      SyncerCreated,
	}

	return cs, nil
}

func (cs *GeneralSyncer) SetLogger(l logging.Logger) logging.Logger {
	if sl, ok := cs.storage.(logging.SetLogger); ok {
		_ = sl.SetLogger(l)
	}

	return cs.Logging.SetLogger(l)
}

func (cs *GeneralSyncer) Close() error {
	if err := cs.storage.Close(); err != nil {
		return err
	}

	return nil
}

func (cs *GeneralSyncer) SetStateChan(stateChan chan<- Syncer) *GeneralSyncer {
	cs.stateChan = stateChan

	return cs
}

func (cs *GeneralSyncer) State() SyncerState {
	cs.RLock()
	defer cs.RUnlock()

	return cs.st
}

func (cs *GeneralSyncer) setState(state SyncerState) {
	cs.Lock()
	defer cs.Unlock()

	cs.Log().Debug().Str("new_state", state.String()).Msg("state changed")

	cs.st = state

	if cs.stateChan != nil {
		go func() {
			cs.stateChan <- cs
		}()
	}
}

func (cs *GeneralSyncer) Save() error {
	if cs.State() == SyncerSaved {
		return nil
	}

	cs.readyToSave()

	if cs.State() != SyncerPrepared {
		return nil
	}

	return cs.save()
}

func (cs *GeneralSyncer) save() error {
	cs.setState(SyncerSaving)
	defer cs.setState(SyncerSaved)

	if err := cs.startBlocks(); err != nil {
		return err
	}

	if err := cs.commit(); err != nil {
		return err
	}

	if err := cs.storage.Close(); err != nil {
		return err
	}

	return nil
}

func (cs *GeneralSyncer) reset() error {
	cs.Lock()
	defer cs.Unlock()

	provedNodes := make([]base.Address, len(cs.sourceNodes))
	{
		var i int
		for k := range cs.sourceNodes {
			provedNodes[i] = k
			i++
		}
	}

	cs.pn = provedNodes
	cs.baseManifest = nil

	if s, err := cs.localstate.Storage().SyncerStorage(); err != nil {
		return err
	} else {
		cs.storage = s
	}

	if sl, ok := cs.storage.(logging.SetLogger); ok {
		_ = sl.SetLogger(cs.Log())
	}

	return nil
}

func (cs *GeneralSyncer) provedNodes() []base.Address {
	cs.RLock()
	defer cs.RUnlock()

	return cs.pn
}

func (cs *GeneralSyncer) Prepare(baseManifest block.Manifest) error {
	if cs.State() >= SyncerPrepared {
		cs.Log().Debug().Msg("already prepared")
		return nil
	}

	cs.Lock()
	cs.baseManifest = baseManifest
	cs.Unlock()

	go func() {
		// NOTE do forever unless successfully done
		_ = util.Retry(0, time.Millisecond*500, func() error {
			if err := cs.reset(); err != nil {
				cs.Log().Error().Err(err).Msg("failed to reset for syncing")

				return err
			}

			if len(cs.provedNodes()) < 1 {
				return xerrors.Errorf("empty proved nodes")
			}

			if err := cs.prepare(); err != nil {
				cs.Log().Error().Err(err).Msg("failed to prepare for syncing")

				return err
			}

			return nil
		})
	}()

	return nil
}

func (cs *GeneralSyncer) prepare() error {
	cs.Log().Debug().Msg("trying to prepare")

	cs.setState(SyncerPreparing)

	if err := cs.headAndTailManifests(); err != nil {
		return err
	}

	if err := cs.fillManifests(); err != nil {
		return err
	}

	cs.setState(SyncerPrepared)

	if cs.isReadyToSave() {
		if err := cs.save(); err != nil {
			return err
		}
	}

	return nil
}

func (cs *GeneralSyncer) isReadyToSave() bool {
	cs.RLock()
	defer cs.RUnlock()

	return cs.willSave
}

func (cs *GeneralSyncer) readyToSave() {
	cs.Lock()
	defer cs.Unlock()

	cs.willSave = true
}

func (cs *GeneralSyncer) headAndTailManifests() error {
	var heights []base.Height
	if cs.heightFrom == cs.heightTo {
		heights = []base.Height{cs.heightFrom}
	} else {
		heights = []base.Height{cs.heightFrom, cs.heightTo}
	}

	var fetched map[base.Address][]block.Manifest
	if bs := cs.fetchManifestsByNodes(heights); len(bs) < 1 {
		return xerrors.Errorf("failed to fetch manifests from all of source nodes")
	} else {
		fetched = bs
	}

	var manifests []block.Manifest
	var provedNodes []base.Address
	switch ms, pn, err := cs.checkThresholdByHeights(heights, fetched); {
	case err != nil:
		return err
	case len(pn) < 1:
		return xerrors.Errorf("empty proved nodes")
	default:
		manifests = ms
		provedNodes = pn
	}

	if cs.baseManifest != nil {
		checker := NewManifestsValidationChecker(cs.localstate, []block.Manifest{cs.baseManifest, manifests[0]})
		_ = checker.SetLogger(cs.Log())

		if err := util.NewChecker("sync-manifests-validation-checker", []util.CheckerFunc{
			checker.CheckSerialized,
		}).Check(); err != nil {
			return err
		}
	}

	for i, height := range heights {
		b := manifests[i]
		if height != b.Height() {
			return xerrors.Errorf("invalid Manifest found; height does not match")
		}
	}

	cs.Lock()
	cs.pn = provedNodes
	cs.Unlock()

	if err := cs.storage.SetManifests(manifests); err != nil {
		return err
	}

	return nil
}

func (cs *GeneralSyncer) fillManifests() error {
	if cs.heightFrom == cs.heightTo || cs.heightTo == cs.heightFrom+1 {
		return nil
	}

	from := (cs.heightFrom + 1).Int64()
	to := cs.heightTo.Int64()
	heights := make([]base.Height, int(to-from))
	for i := from; i < to; i++ {
		heights[i-from] = base.Height(i)
	}

	var fetched map[base.Address][]block.Manifest
	if bs := cs.fetchManifestsByNodes(heights); len(bs) < 1 {
		return xerrors.Errorf("failed to fetch manifests from all of source nodes")
	} else {
		fetched = bs
	}

	switch ms, pn, err := cs.checkThresholdByHeights(heights, fetched); {
	case err != nil:
		return err
	case len(pn) < 1:
		return xerrors.Errorf("empty proved nodes")
	default:
		for i, height := range heights {
			b := ms[i]
			if height != b.Height() {
				return xerrors.Errorf("invalid Manifest found; height does not match")
			}
		}

		cs.Lock()
		cs.pn = pn
		cs.Unlock()

		if err := cs.storage.SetManifests(ms); err != nil {
			return err
		}
	}

	return nil
}

func (cs *GeneralSyncer) startBlocks() error {
	cs.Log().Debug().Msg("start to fetch blocks")
	defer cs.Log().Debug().Msg("fetched blocks")

	_ = util.Retry(0, time.Second, func() error {
		err := cs.fetchBlocksByNodes()
		if err != nil {
			cs.Log().Error().Err(err).Msg("failed to fetch blocks by nodes")
		}

		return err
	})

	return nil
}

func (cs *GeneralSyncer) fetchBlocksByNodes() error {
	cs.Log().Debug().Msg("start to fetch blocks by nodes")

	worker := util.NewWorker("sync-fetch-blocks", 10)
	defer worker.Done()
	_ = worker.SetLogger(cs.Log())

	for _, address := range cs.provedNodes() {
		node := cs.sourceNodes[address]
		worker.Run(cs.workerCallbackFetchBlocks(node))
	}

	if err := cs.distributeBlocksJob(worker); err != nil {
		return err
	}

	var received uint
	for err := range worker.Errors() {
		received++
		if err = cs.handleSyncerFetchBlockError(err); err != nil {
			return err
		}

		if received == worker.Jobs() {
			break
		}
	}

	cs.Log().Debug().Msg("fetched blocks by nodes")

	// check fetched blocks
	for i := cs.heightFrom.Int64(); i <= cs.heightTo.Int64(); i++ {
		if found, err := cs.storage.HasBlock(base.Height(i)); err != nil {
			return xerrors.Errorf("some block not found after fetching blocks: height=%d; %w", i, err)
		} else if !found {
			return xerrors.Errorf("some block not found after fetching blocks: height=%d", i)
		}
	}

	return nil
}

func (cs *GeneralSyncer) handleSyncerFetchBlockError(err error) error {
	if err == nil {
		return nil
	}

	var fm *syncerFetchBlockError
	if !xerrors.As(err, &fm) {
		cs.Log().Error().Err(err).Msg("something wrong to fetch blocks")
		return nil
	}

	if fm.err != nil {
		cs.Log().Error().Err(err).
			Hinted("source_node", fm.node).Msg("something wrong to fetch blocks from node")

		return xerrors.Errorf("failed to fetch blocks; %w", fm.err)
	}

	if len(fm.blocks) < 1 {
		cs.Log().Error().Err(err).
			Hinted("source_node", fm.node).Msg("empty blocks; something wrong to fetch blocks from node")

		return xerrors.Errorf("empty blocks; failed to fetch blocks")
	}

	if ms, err := cs.checkFetchedBlocks(fm.blocks); err != nil {
		return err
	} else if len(fm.missing) > 0 || len(ms) > 0 {
		cs.Log().Error().Interface("missing_blocks", len(fm.missing)+len(ms)).Msg("still missing blocks found")

		return xerrors.Errorf("some missing blocks found; failed to fetch blocks")
	}

	return nil
}

func (cs *GeneralSyncer) distributeBlocksJob(worker *util.Worker) error {
	from := cs.heightFrom.Int64()
	to := cs.heightTo.Int64()

	limit := cs.limitBlocksPerOnce
	{ // more widely distribute requests
		total := int(to - from)
		if total < len(cs.provedNodes())*limit {
			limit = total / len(cs.provedNodes())
		}
	}

	var heights []base.Height
	for i := from; i <= to; i++ {
		if found, err := cs.storage.HasBlock(base.Height(i)); err != nil {
			return err
		} else if found {
			continue
		}

		heights = append(heights, base.Height(i))
		if len(heights) == limit {
			worker.NewJob(heights)

			heights = nil
		}
	}

	if len(heights) > 0 {
		worker.NewJob(heights)
	}

	return nil
}

func (cs *GeneralSyncer) fetchManifestsByNodes(heights []base.Height) map[base.Address][]block.Manifest {
	cs.Log().Debug().
		Hinted("height_from", heights[0]).
		Hinted("height_to", heights[len(heights)-1]).
		Msg("trying to fetch manifest")

	resultChan := make(chan map[base.Address][]block.Manifest, len(cs.provedNodes()))

	for _, address := range cs.provedNodes() {
		go func(address base.Address) {
			manifests := cs.callbackFetchManifests(cs.sourceNodes[address], heights)
			resultChan <- map[base.Address][]block.Manifest{address: manifests}
		}(address)
	}

	fetched := map[base.Address][]block.Manifest{}
	for result := range resultChan {
		for address, manifests := range result {
			fetched[address] = manifests
		}

		if len(fetched) == len(cs.provedNodes()) {
			break
		}
	}

	cs.Log().Debug().Int("fetched", len(fetched)).Msg("fetched manifests")

	return fetched
}

func (cs *GeneralSyncer) callbackFetchManifests(node Node, heights []base.Height) []block.Manifest {
	manifests := make([]block.Manifest, len(heights))

	updateManifests := func(fetched []block.Manifest) {
		sort.SliceStable(fetched, func(i, j int) bool {
			return fetched[i].Height() < fetched[j].Height()
		})

		var last int
		for i := range fetched {
			b := fetched[i]
			for j := range heights[last:] {
				if b.Height() != heights[last+j] {
					continue
				}

				manifests[last+j] = b
				last = j + 1
				break
			}
		}
	}

	var sliced []base.Height // nolint
	for i := range heights {
		height := heights[i]
		sliced = append(sliced, height)
		if len(sliced) != cs.limitManifestsPerWorker {
			continue
		}

		fetched := cs.callbackFetchManifestsSlice(node, sliced)
		updateManifests(fetched)
	}

	if len(sliced) > 0 {
		fetched := cs.callbackFetchManifestsSlice(node, sliced)
		updateManifests(fetched)
	}

	return manifests
}

func (cs *GeneralSyncer) callbackFetchManifestsSlice(node Node, heights []base.Height) []block.Manifest {
	var retries uint = 3

	l := cs.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Uint("retries", retries).
			Hinted("source_node", node.Address()).
			Interface("heights", heights)
	})

	l.Debug().Msg("trying to fetch manifest of node")

	var manifests []block.Manifest

	missing := heights
	_ = util.Retry(retries, time.Millisecond*300, func() error { // TODO retry count should be configurable
		bs, err := cs.fetchManifests(node, missing)
		if err != nil {
			return err
		}

		if ss, ms, err := cs.sanitizeManifests(heights, bs); err != nil {
			return err
		} else {
			manifests = ss
			missing = ms
		}

		if len(missing) > 0 {
			return xerrors.Errorf("something missing")
		}

		return nil
	})

	l.Debug().Interface("missing", missing).Int("fetched", len(manifests)).Msg("fetched manifest of node")

	return manifests
}

func (cs *GeneralSyncer) checkThresholdByHeights(heights []base.Height, fetched map[base.Address][]block.Manifest) (
	[]block.Manifest, // major manifests
	[]base.Address, // nodes, which have over threshold manifests
	error,
) {
	threshold := cs.localstate.Policy().Threshold()
	manifests := make([]block.Manifest, len(heights))

	var pn []base.Address = cs.provedNodes()
	for index := range heights {
		provedNodes := map[base.Address]Node{}
		{
			for i := range pn {
				node := cs.sourceNodes[pn[i]]
				provedNodes[node.Address()] = node
			}
		}

		if m, p, err := cs.checkThreshold(index, heights, fetched, provedNodes, threshold); err != nil {
			return nil, nil, err
		} else {
			manifests[index] = m
			pn = p
		}
	}

	return manifests, pn, nil
}

func (cs *GeneralSyncer) checkThreshold(
	index int,
	heights []base.Height,
	fetched map[base.Address][]block.Manifest,
	provedNodes map[base.Address]Node,
	threshold base.Threshold,
) (block.Manifest, []base.Address, error) {
	height := heights[index]
	hashByNode := map[string][]base.Address{}
	ms := map[string]block.Manifest{}

	var set []string // nolint
	for node := range fetched {
		bs := fetched[node]
		if len(bs) != len(heights) {
			cs.Log().Debug().
				Int("expected", len(heights)).
				Int("returned", len(bs)).
				Msg("failed to get the expected data from node")

			continue
		}

		if len(provedNodes) > 0 {
			if _, found := provedNodes[node]; !found {
				continue
			}
		}

		if bs[index] == nil {
			continue
		}

		key := bs[index].Hash().String()
		set = append(set, key)
		ms[key] = bs[index]
		hashByNode[key] = append(hashByNode[key], node)
	}
	result, key := base.FindMajorityFromSlice(threshold.Total, threshold.Threshold, set)

	if cs.Log().IsVerbose() {
		var ns []string
		for n := range provedNodes {
			ns = append(ns, provedNodes[n].Address().String())
		}

		cs.Log().Debug().
			Str("result", result.String()).
			Str("majority_block_hash", key).
			Hinted("height", height).
			Strs("target_nodes", ns).
			Msg("check majority of manifests")
	}

	if result != base.VoteResultMajority {
		return nil, nil, xerrors.Errorf("given target nodes doet not have common blocks: height=%s", height)
	}

	return ms[key], hashByNode[key], nil
}

func (cs *GeneralSyncer) fetchManifests(node Node, heights []base.Height) ([]block.Manifest, error) { // nolint
	l := cs.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("source_node", node.Address()).
			Hinted("height_from", heights[0]).
			Hinted("height_to", heights[len(heights)-1])
	})

	l.Debug().Msg("trying to fetch manifests")

	var fetched []block.Manifest
	if bs, err := node.Channel().Manifests(heights); err != nil {
		return nil, err
	} else {
		sort.SliceStable(bs, func(i, j int) bool {
			return bs[i].Height() < bs[j].Height()
		})
		fetched = bs
	}

	l.Debug().Int("fetched", len(fetched)).Msg("fetched manifests")

	return fetched, nil
}

// sanitizeManifests checks and filter the fetched Manifests. NOTE the
// input heights should be sorted by it's Height.
func (cs *GeneralSyncer) sanitizeManifests(heights []base.Height, l interface{}) (
	[]block.Manifest, []base.Height, error,
) {
	var bs []block.Manifest
	switch t := l.(type) {
	case []block.Block:
		for _, b := range t {
			bs = append(bs, b)
		}
	case []block.Manifest:
		bs = t
	default:
		return nil, nil, xerrors.Errorf("not Manifest like: %T", l)
	}

	var checked []block.Manifest
	var missing []base.Height
	{
		head := heights[0]
		tail := heights[len(heights)-1]

		a := map[base.Height]block.Manifest{}
		for _, i := range bs {
			b := i.(block.Manifest)
			if b.Height() < head || b.Height() > tail {
				continue
			} else if _, found := a[b.Height()]; found {
				continue
			}

			a[b.Height()] = b
		}

		for _, h := range heights {
			if b, found := a[h]; !found {
				missing = append(missing, h)
			} else {
				checked = append(checked, b)
			}
		}
	}

	return checked, missing, nil
}

func (cs *GeneralSyncer) workerCallbackFetchBlocks(node Node) util.WorkerCallback {
	return func(jobID uint, job interface{}) error {
		var heights []base.Height
		if h, ok := job.([]base.Height); !ok {
			return xerrors.Errorf("job is not []Height: %T", job)
		} else {
			heights = h
		}

		l := cs.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
			return ctx.Hinted("source_node", node.Address()).
				Interface("heights", heights)
		})

		l.Debug().Msg("trying to fetch blocks")
		defer l.Debug().Msg("fetched blocks")

		var manifests []block.Manifest
		var missing []base.Height
		var err error
		if bs, e := cs.fetchBlocks(node, heights); err != nil {
			err = e
		} else if manifests, missing, err = cs.sanitizeManifests(heights, bs); err != nil {
		}

		blocks := make([]block.Block, len(manifests))
		for i := range manifests {
			blocks[i] = manifests[i].(block.Block)
		}

		return &syncerFetchBlockError{
			node:    node.Address(),
			heights: heights,
			err:     err,
			blocks:  blocks,
			missing: missing,
		}
	}
}

func (cs *GeneralSyncer) checkFetchedBlocks(fetched []block.Block) ([]base.Height, error) {
	networkID := cs.localstate.Policy().NetworkID()

	var filtered []block.Block // nolint
	var missing []base.Height
	for i := range fetched {
		blk := fetched[i].(block.Block)
		if err := blk.IsValid(networkID); err != nil {
			missing = append(missing, blk.Height())
			continue
		}

		if manifest, err := cs.storage.Manifest(blk.Height()); err != nil {
			return nil, err
		} else if !manifest.Hash().Equal(blk.Hash()) {
			missing = append(missing, blk.Height())
			continue
		}

		filtered = append(filtered, blk)
	}

	if err := cs.storage.SetBlocks(filtered); err != nil {
		return nil, err
	}

	return missing, nil
}

func (cs *GeneralSyncer) fetchBlocks(node Node, heights []base.Height) ([]block.Block, error) { // nolint
	l := cs.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("source_node", node.Address()).
			Hinted("height_from", heights[0]).
			Hinted("height_to", heights[len(heights)-1])
	})

	l.Debug().Msg("trying to fetch blocks")

	var fetched []block.Block
	if bs, err := node.Channel().Blocks(heights); err != nil {
		return nil, err
	} else {
		sort.SliceStable(bs, func(i, j int) bool {
			return bs[i].Height() < bs[j].Height()
		})
		fetched = bs
	}

	l.Debug().Int("fetched", len(fetched)).Msg("fetched blocks")

	return fetched, nil
}

func (cs *GeneralSyncer) commit() error {
	return cs.storage.Commit()
}

func (cs *GeneralSyncer) HeightFrom() base.Height {
	return cs.heightFrom
}

func (cs *GeneralSyncer) HeightTo() base.Height {
	return cs.heightTo
}

func (cs *GeneralSyncer) TailManifest() block.Manifest {
	b, err := cs.storage.Manifest(cs.heightTo)
	if err != nil {
		return nil
	}

	return b
}
