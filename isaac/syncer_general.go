package isaac

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/errors"
	"github.com/spikeekips/mitum/util/logging"
)

var blockIntegrityError = errors.NewError("block integrity failed")

type BlockIntegrityError struct {
	*errors.NError
	From block.Manifest
	Err  error
}

func NewBlockIntegrityError(from block.Manifest, err error) *BlockIntegrityError {
	return &BlockIntegrityError{
		NError: blockIntegrityError,
		From:   from,
		Err:    err,
	}
}

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
	ost                     storage.Database
	blockData               blockdata.BlockData
	policy                  *LocalPolicy
	st                      storage.SyncerSession
	sourceNodes             map[base.Address]network.Node
	heightFrom              base.Height
	heightTo                base.Height
	limitManifestsPerWorker int
	limitBlocksPerOnce      int
	pn                      []base.Address
	state                   SyncerState
	baseManifest            block.Manifest
	stateChan               chan<- SyncerStateChangedContext
	tailManifest            block.Manifest
	blocks                  []block.Block
	blockDataSessions       []blockdata.Session
	lifeCtx                 context.Context
	lifeCancel              func()
}

func NewGeneralSyncer(
	local *network.LocalNode,
	ost storage.Database,
	blockData blockdata.BlockData,
	policy *LocalPolicy,
	sourceNodes []network.Node,
	baseManifest block.Manifest,
	to base.Height,
) (*GeneralSyncer, error) {
	var from base.Height
	if baseManifest == nil {
		from = base.PreGenesisHeight
	} else {
		from = baseManifest.Height() + 1
	}

	switch {
	case from > to:
		return nil, xerrors.Errorf("from height, %d is greater than to height, %d", from, to)
	case len(sourceNodes) < 1:
		return nil, xerrors.Errorf("empty source nodes")
	}

	if m, found, err := ost.LastManifest(); err != nil {
		return nil, err
	} else if found && from <= m.Height() {
		return nil, xerrors.Errorf("from height is same or lower than last block; from=%d last=%d", from, m.Height())
	}

	var sn map[base.Address]network.Node
	if i, err := validateSyncerSourceNodes(local, sourceNodes); err != nil {
		return nil, err
	} else {
		sn = i
	}

	cs := &GeneralSyncer{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.
				Hinted("from", from).Hinted("to", to).
				Str("syncer_id", util.UUID().String()).
				Str("module", "general-syncer")
		}),
		ost:                     ost,
		blockData:               blockData,
		policy:                  policy,
		sourceNodes:             sn,
		heightFrom:              from,
		heightTo:                to,
		baseManifest:            baseManifest,
		limitManifestsPerWorker: 10,
		limitBlocksPerOnce:      10,
		state:                   SyncerCreated,
		blocks:                  make([]block.Block, to-from+1),
		blockDataSessions:       make([]blockdata.Session, to-from+1),
	}
	cs.initializeProvedNodes()
	cs.lifeCtx, cs.lifeCancel = context.WithCancel(context.Background())

	return cs, nil
}

func (cs *GeneralSyncer) SetLogger(l logging.Logger) logging.Logger {
	if sl, ok := cs.database().(logging.SetLogger); ok {
		_ = sl.SetLogger(l)
	}

	return cs.Logging.SetLogger(l)
}

func (cs *GeneralSyncer) ID() string {
	return fmt.Sprintf("%v-%v", cs.heightFrom, cs.heightTo)
}

func (cs *GeneralSyncer) Close() error {
	cs.Lock()
	defer cs.Unlock()

	if cs.st == nil {
		return nil
	}

	if cs.lifeCancel != nil {
		cs.lifeCancel()
	}

	defer cs.Log().Debug().Msg("closed")

	return cs.st.Close()
}

func (cs *GeneralSyncer) SetStateChan(stateChan chan<- SyncerStateChangedContext) *GeneralSyncer {
	cs.stateChan = stateChan

	return cs
}

func (cs *GeneralSyncer) State() SyncerState {
	cs.RLock()
	defer cs.RUnlock()

	return cs.state
}

func (cs *GeneralSyncer) HeightFrom() base.Height {
	return cs.heightFrom
}

func (cs *GeneralSyncer) HeightTo() base.Height {
	return cs.heightTo
}

func (cs *GeneralSyncer) setTailManifest(m block.Manifest) {
	cs.Lock()
	defer cs.Unlock()

	cs.tailManifest = m
}

func (cs *GeneralSyncer) TailManifest() block.Manifest {
	cs.RLock()
	defer cs.RUnlock()

	return cs.tailManifest
}

func (cs *GeneralSyncer) Prepare() error {
	cs.Lock()
	defer cs.Unlock()

	if cs.state >= SyncerPrepared {
		cs.Log().Debug().Msg("already prepared")

		return nil
	}

	go func() {
		// NOTE do forever unless successfully done
	end:
		for {
			select {
			case <-cs.lifeCtx.Done():
				break end
			default:
				err := cs.prepare()
				if err == nil {
					break end
				}

				cs.Log().Error().Err(err).Msg("failed to prepare for syncing")

				var rollbackCtx *BlockIntegrityError
				if xerrors.As(err, &rollbackCtx) {
					if err := cs.rollback(rollbackCtx); err != nil {
						cs.Log().Error().Err(err).Msg("failed to rollback")

						<-time.After(time.Millisecond * 500)

						continue
					} else {
						break end
					}
				}

				<-time.After(time.Millisecond * 500)
			}
		}
	}()

	return nil
}

func (cs *GeneralSyncer) prepare() error {
	cs.Log().Debug().Msg("trying to prepare")

	if err := cs.reset(); err != nil {
		cs.Log().Error().Err(err).Msg("failed to reset for syncing")

		return err
	}

	cs.setState(SyncerPreparing, false)

	if len(cs.provedNodes()) < 1 {
		return xerrors.Errorf("empty proved nodes")
	}

	if cs.State() < SyncerPrepared {
		if err := cs.headAndTailManifests(); err != nil {
			return err
		}

		if err := cs.fillManifests(); err != nil {
			return err
		}
	}

	cs.Log().Debug().Msg("prepared")

	cs.setState(SyncerPrepared, false)

	return cs.save()
}

func (cs *GeneralSyncer) save() error {
	if cs.State() != SyncerPrepared {
		cs.Log().Debug().Msg("not yet prepared")

		return nil
	} else if cs.State() == SyncerSaved {
		cs.Log().Debug().Msg("already saved")

		return nil
	}

	cs.Log().Debug().Msg("trying to save")

	cs.setState(SyncerSaving, false)

	if err := cs.startBlocks(); err != nil {
		return err
	}

	if err := cs.commit(); err != nil {
		return err
	}

	cs.Log().Debug().Msg("saved")

	cs.setState(SyncerSaved, false)

	return nil
}

func (cs *GeneralSyncer) reset() error {
	cs.Log().Debug().Msg("syncer will be reset")

	cs.setState(SyncerCreated, true)

	cs.Lock()
	defer cs.Unlock()

	if cs.st != nil {
		cs.Log().Debug().Int("blocks", len(cs.blocks)).Msg("reset: will cleanup storage; database and block data")
		if err := blockdata.CleanByHeight(cs.ost, cs.blockData, cs.heightFrom); err != nil {
			return err
		}

		// NOTE clean up blocks in block data session
		cs.Log().Debug().Int("blocks", len(cs.blocks)).Msg("reset: will cleanup block data session")
		for i := range cs.blockDataSessions {
			ss := cs.blockDataSessions[i]
			if ss == nil {
				continue
			}

			if err := ss.Cancel(); err != nil {
				return err
			}
		}

		if err := cs.st.Close(); err != nil {
			return err
		}
	}

	cs.blockDataSessions = make([]blockdata.Session, cs.heightTo-cs.heightFrom+1)
	cs.blocks = make([]block.Block, cs.heightTo-cs.heightFrom+1)

	if st, err := cs.ost.NewSyncerSession(); err != nil {
		return err
	} else {
		cs.st = st
	}

	if sl, ok := cs.st.(logging.SetLogger); ok {
		_ = sl.SetLogger(cs.Log())
	}

	cs.initializeProvedNodes()

	return nil
}

func (cs *GeneralSyncer) headAndTailManifests() error {
	if cs.State() != SyncerPreparing {
		cs.Log().Debug().Str("state", cs.State().String()).Msg("not preparing state")

		return nil
	}

	var heights []base.Height
	if cs.heightFrom == cs.heightTo {
		heights = []base.Height{cs.heightFrom}
	} else {
		heights = []base.Height{cs.heightFrom, cs.heightTo}
	}

	var manifests []block.Manifest
	var provedNodes []base.Address
	switch ms, pn, err := cs.fetchManifestsByNodes(heights); {
	case err != nil:
		return err
	case len(ms) < 1:
		return xerrors.Errorf("failed to fetch manifests from all of source nodes")
	default:
		manifests = ms
		provedNodes = pn
	}

	if cs.baseManifest != nil {
		head := manifests[0]
		cs.Log().Debug().
			Hinted("base_manifest_previous", cs.baseManifest.PreviousBlock()).
			Hinted("base_manifest", cs.baseManifest.Hash()).
			Hinted("head_previous", head.PreviousBlock()).
			Hinted("head", head.Hash()).
			Msg("checking base and head manifest")

		checker := NewManifestsValidationChecker(cs.policy.NetworkID(), []block.Manifest{cs.baseManifest, head})
		_ = checker.SetLogger(cs.Log())

		if err := util.NewChecker("sync-manifests-validation-checker", []util.CheckerFunc{
			checker.CheckSerialized,
		}).Check(); err != nil {
			cs.Log().Error().Err(err).Msg("failed to verify manifests")
			return err
		}
	}

	cs.setProvedNodes(provedNodes)

	if err := cs.database().SetManifests(manifests); err != nil {
		return err
	}

	cs.setTailManifest(manifests[len(manifests)-1])

	return nil
}

func (cs *GeneralSyncer) fillManifests() error {
	if cs.State() != SyncerPreparing {
		cs.Log().Debug().Str("state", cs.State().String()).Msg("not preparing state")

		return nil
	}

	if cs.heightFrom == cs.heightTo || cs.heightTo == cs.heightFrom+1 {
		return nil
	}

	fill := func(heights []base.Height) error {
		switch ms, pn, err := cs.fetchManifestsByNodes(heights); {
		case err != nil:
			return err
		case len(ms) < 1:
			return xerrors.Errorf("failed to fetch manifests from all of source nodes")
		case len(pn) < 1:
			return xerrors.Errorf("empty proved nodes")
		default:
			cs.setProvedNodes(pn)

			return cs.database().SetManifests(ms)
		}
	}

	from := cs.heightFrom + 1
	to := cs.heightTo

	var heights []base.Height
	for i := from; i < to; i++ {
		heights = append(heights, i)
		if len(heights) != cs.limitBlocksPerOnce {
			continue
		}

		if err := fill(heights); err != nil {
			return err
		} else {
			heights = nil
		}
	}

	if len(heights) > 0 {
		if err := fill(heights); err != nil {
			return err
		}
	}

	return nil
}

func (cs *GeneralSyncer) startBlocks() error {
	if cs.State() != SyncerSaving {
		return xerrors.Errorf("not saving state: %v", cs.State())
	}

	cs.Log().Debug().Msg("start to fetch blocks")
	defer cs.Log().Debug().Msg("fetched blocks")

end:
	for {
		select {
		case <-cs.lifeCtx.Done():
			break end
		default:
			if err := cs.fetchBlocksByNodes(); err != nil {
				cs.Log().Error().Err(err).Msg("failed to fetch blocks by nodes")

				<-time.After(time.Millisecond * 500)
			}

			break end
		}
	}

	return nil
}

func (cs *GeneralSyncer) fetchBlocksByNodes() error {
	cs.Log().Debug().Msg("start to fetch blocks by nodes")

	worker := util.NewParallelWorker("sync-fetch-blocks", 5)
	defer worker.Done()
	_ = worker.SetLogger(cs.Log())

	if len(cs.provedNodes()) < 1 {
		return xerrors.Errorf("empty proved nodes")
	}

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
	for i := cs.heightFrom; i <= cs.heightTo; i++ {
		if found, err := cs.database().HasBlock(i); err != nil {
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

func (cs *GeneralSyncer) distributeBlocksJob(worker *util.ParallelWorker) error {
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
		if found, err := cs.database().HasBlock(base.Height(i)); err != nil {
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

func (cs *GeneralSyncer) fetchManifestsByNodes(heights []base.Height) (
	[]block.Manifest, []base.Address, error,
) {
	l := cs.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("height_from", heights[0]).
			Hinted("height_to", heights[len(heights)-1])
	})

	l.Debug().Msg("trying to fetch manifest")

	resultChan := make(chan map[base.Address][]block.Manifest, len(cs.provedNodes()))

	provedNodes := cs.provedNodes()
	var wg sync.WaitGroup
	wg.Add(len(provedNodes))

	for _, address := range provedNodes {
		go func(address base.Address) {
			defer wg.Done()

			if i, err := cs.callbackFetchManifests(cs.sourceNodes[address], heights); err != nil {
				l.Error().Err(err).Hinted("node", address).Msg("failed to get manifest from node")
				resultChan <- nil
			} else {
				resultChan <- map[base.Address][]block.Manifest{address: i}
			}
		}(address)
	}

	wg.Wait()
	close(resultChan)

	fetched := map[base.Address][]block.Manifest{}
	for result := range resultChan {
		if len(result) < 1 {
			continue
		}

		for address, manifests := range result {
			fetched[address] = manifests
		}
	}

	cs.Log().Debug().Int("fetched", len(fetched)).Msg("fetched manifests")

	switch ms, pn, err := cs.checkThresholdByHeights(heights, fetched); {
	case err != nil:
		return nil, nil, err
	case len(pn) < 1:
		return nil, nil, xerrors.Errorf("empty proved nodes")
	default:
		for i, height := range heights {
			b := ms[i]
			if height != b.Height() {
				return nil, nil, xerrors.Errorf("invalid Manifest found; height does not match")
			}
		}

		return ms, pn, nil
	}
}

func (cs *GeneralSyncer) callbackFetchManifests(node network.Node, heights []base.Height) ([]block.Manifest, error) {
	manifests := make([]block.Manifest, len(heights))

	update := func(fetched []block.Manifest) {
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

		if i, err := cs.callbackFetchManifestsSlice(node, sliced); err != nil {
			return nil, err
		} else {
			update(i)
			sliced = nil
		}
	}

	if len(sliced) > 0 {
		if i, err := cs.callbackFetchManifestsSlice(node, sliced); err != nil {
			return nil, err
		} else {
			update(i)
		}
	}

	return manifests, nil
}

func (cs *GeneralSyncer) callbackFetchManifestsSlice(
	node network.Node, heights []base.Height,
) ([]block.Manifest, error) {
	var maxRetries uint = 3

	l := cs.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Uint("max-retries", maxRetries).
			Hinted("source_node", node.Address()).
			Interface("heights", heights)
	})

	l.Debug().Msg("trying to fetch manifest of node")

	var manifests []block.Manifest

	missing := heights

	if err := util.Retry(maxRetries, time.Millisecond*300, func(retries int) error {
		l.Debug().Int("retries", retries).Msg("try to fetch manifest")

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
	}); err != nil {
		return nil, err
	}

	l.Debug().Interface("missing", missing).Int("fetched", len(manifests)).Msg("fetched manifest of node")

	return manifests, nil
}

func (cs *GeneralSyncer) checkThresholdByHeights(heights []base.Height, fetched map[base.Address][]block.Manifest) (
	[]block.Manifest, // major manifests
	[]base.Address, // nodes, which have over threshold manifests
	error,
) {
	manifests := make([]block.Manifest, len(heights))

	var pn []base.Address = cs.provedNodes()
	for index := range heights {
		provedNodes := map[base.Address]network.Node{}
		{
			for i := range pn {
				node := cs.sourceNodes[pn[i]]
				provedNodes[node.Address()] = node
			}
		}

		if m, p, err := cs.checkThreshold(index, heights, fetched, provedNodes); err != nil {
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
	provedNodes map[base.Address]network.Node,
) (block.Manifest, []base.Address, error) {
	height := heights[index]
	hashByNode := map[string][]base.Address{}
	ms := map[string]block.Manifest{}

	var set []string // nolint
	for node := range fetched {
		bs := fetched[node]
		if len(bs) != len(heights) {
			cs.Log().Debug().Int("expected", len(heights)).Int("returned", len(bs)).
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

	if len(set) < 1 {
		return nil, nil, xerrors.Errorf("nothing fetched for height=%d", height)
	}

	var threshold base.Threshold
	if t, err := base.NewThreshold(uint(len(set)), cs.policy.ThresholdRatio()); err != nil {
		return nil, nil, err
	} else {
		threshold = t
	}

	result, key := base.FindMajorityFromSlice(threshold.Total, threshold.Threshold, set)
	if result != base.VoteResultMajority {
		return nil, nil, xerrors.Errorf("given target nodes doet not have common blocks: height=%s", height)
	}

	return ms[key], hashByNode[key], nil
}

func (cs *GeneralSyncer) fetchManifests(node network.Node, heights []base.Height) ([]block.Manifest, error) { // nolint
	var maps []block.BlockDataMap
	if i, err := cs.fetchBlockDataMaps(node, heights); err != nil {
		return nil, err
	} else {
		maps = i
	}

	resultchan := make(chan interface{}, len(heights))

	var wg sync.WaitGroup
	wg.Add(len(maps))
	for i := range maps {
		go func(bd block.BlockDataMap) {
			defer wg.Done()
			if i, err := node.Channel().BlockData(context.Background(), bd.Manifest()); err != nil {
				resultchan <- err
			} else {
				defer func() {
					_ = i.Close()
				}()

				if j, err := cs.blockData.Writer().ReadManifest(i); err != nil {
					resultchan <- err
				} else if err := block.CompareManifestWithMap(j, bd); err != nil {
					resultchan <- err
				} else {
					resultchan <- j
				}
			}
		}(maps[i])
	}

	wg.Wait()
	close(resultchan)

	var fetched []block.Manifest
	for i := range resultchan {
		switch t := i.(type) {
		case block.Manifest:
			fetched = append(fetched, t)
		case error:
			return nil, xerrors.Errorf("failed to fetch manifest: %w", t)
		default:
			return nil, xerrors.Errorf("failed to fetch manifest; unknown problem")
		}
	}

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
		for i := range bs {
			b := bs[i]
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

func (cs *GeneralSyncer) workerCallbackFetchBlocks(node network.Node) util.WorkerCallback {
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

		var manifests []block.Manifest
		var missing []base.Height
		var err error
		if bs, e := cs.fetchBlocks(node, heights); err != nil {
			err = e
		} else if manifests, missing, err = cs.sanitizeManifests(heights, bs); err != nil {
			err = e
		}

		if err != nil {
			l.Error().Err(err).Msg("failed to fetch blocks")

			return &syncerFetchBlockError{
				node:    node.Address(),
				heights: heights,
				err:     err,
			}
		}

		blocks := make([]block.Block, len(manifests))
		for i := range manifests {
			blocks[i] = manifests[i].(block.Block)
		}
		l.Debug().Int("blocks", len(blocks)).Msg("fetched blocks")

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
	networkID := cs.policy.NetworkID()

	var filtered []block.Block // nolint
	var sessions []blockdata.Session
	var missing []base.Height
	for i := range fetched {
		blk := fetched[i]
		if err := blk.IsValid(networkID); err != nil {
			cs.Log().Error().Err(err).
				Hinted("height", blk.Height()).
				Interface("block", blk).
				Msg("found invalid block")

			missing = append(missing, blk.Height())

			continue
		}

		switch manifest, found, err := cs.database().Manifest(blk.Height()); {
		case !found:
			return nil, util.NotFoundError.Errorf("manifest not found")
		case err != nil:
			return nil, err
		case !manifest.Hash().Equal(blk.Hash()):
			missing = append(missing, blk.Height())

			continue
		}

		if ss := cs.blockDataSessions[blk.Height()-cs.heightFrom]; ss == nil {
			missing = append(missing, blk.Height())

			continue
		} else {
			sessions = append(sessions, ss)
		}

		filtered = append(filtered, blk)
	}

	if len(missing) > 0 {
		return missing, nil
	}

	cs.Lock()
	for i := range filtered {
		blk := filtered[i]
		cs.blocks[blk.Height()-cs.heightFrom] = blk
	}
	cs.Unlock()

	if maps, err := cs.saveBlockData(sessions); err != nil {
		return nil, err
	} else if err := cs.database().SetBlocks(filtered, maps); err != nil {
		return nil, err
	} else {
		return nil, nil
	}
}

func (cs *GeneralSyncer) fetchBlocks(node network.Node, heights []base.Height) ([]block.Block, error) {
	l := cs.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("source_node", node.Address()).
			Hinted("height_from", heights[0]).
			Hinted("height_to", heights[len(heights)-1])
	})

	var maps []block.BlockDataMap
	if i, err := cs.fetchBlockDataMaps(node, heights); err != nil {
		return nil, err
	} else {
		maps = i
	}

	l.Debug().Msg("trying to fetch blocks")

	fetched := make([]block.Block, len(heights))
	for i := range maps {
		if j, err := cs.fetchBlock(node, maps[i]); err != nil {
			l.Error().Err(err).Msg("failed to fetch block")

			return nil, err
		} else {
			fetched[i] = j
		}
	}

	sort.SliceStable(fetched, func(i, j int) bool {
		return fetched[i].Height() < fetched[j].Height()
	})

	l.Debug().Int("fetched", len(fetched)).Msg("fetched blocks")

	return fetched, nil
}

func (cs *GeneralSyncer) blockDataSession(height base.Height) (blockdata.Session, error) {
	cs.Lock()
	defer cs.Unlock()

	if ss := cs.blockDataSessions[height-cs.heightFrom]; ss != nil {
		return ss, nil
	}

	if i, err := cs.blockData.NewSession(height); err != nil {
		return nil, err
	} else {
		cs.blockDataSessions[height-cs.heightFrom] = i

		return i, nil
	}
}

func (cs *GeneralSyncer) fetchBlock(node network.Node, bd block.BlockDataMap) (block.Block, error) { // nolint:funlen
	var ss blockdata.Session
	if i, err := cs.blockDataSession(bd.Height()); err != nil {
		return nil, err
	} else {
		ss = i
	}

	l := cs.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("source_node", node.Address()).
			Hinted("height", bd.Height())
	})

	l.Debug().Msg("trying to fetch block")

	blk := (interface{})(block.BlockV0{}).(block.BlockUpdater)

	switch i, found, err := cs.database().Manifest(bd.Height()); {
	case err != nil:
		return nil, err
	case !found:
		return nil, util.NotFoundError.Errorf("manifest, %d not found", bd.Height())
	default:
		if err := ss.SetManifest(i); err != nil {
			return nil, err
		}

		blk = blk.SetManifest(i)
	}

	if i, err := cs.fetchBlockData(node, bd.Operations(), ss); err != nil {
		return nil, err
	} else if j, err := cs.blockData.Writer().ReadOperations(i); err != nil {
		return nil, err
	} else if j != nil {
		blk = blk.SetOperations(j)
	}

	if i, err := cs.fetchBlockData(node, bd.OperationsTree(), ss); err != nil {
		return nil, err
	} else if j, err := cs.blockData.Writer().ReadOperationsTree(i); err != nil {
		return nil, err
	} else if j.Len() > 0 {
		blk = blk.SetOperationsTree(j)
	}

	if i, err := cs.fetchBlockData(node, bd.States(), ss); err != nil {
		return nil, err
	} else if j, err := cs.blockData.Writer().ReadStates(i); err != nil {
		return nil, err
	} else if j != nil {
		blk = blk.SetStates(j)
	}

	if i, err := cs.fetchBlockData(node, bd.StatesTree(), ss); err != nil {
		return nil, err
	} else if j, err := cs.blockData.Writer().ReadStatesTree(i); err != nil {
		return nil, err
	} else if j.Len() > 0 {
		blk = blk.SetStatesTree(j)
	}

	if i, err := cs.fetchBlockData(node, bd.INITVoteproof(), ss); err != nil {
		return nil, err
	} else if j, err := cs.blockData.Writer().ReadINITVoteproof(i); err != nil {
		return nil, err
	} else if j != nil {
		blk = blk.SetINITVoteproof(j)
	}

	if i, err := cs.fetchBlockData(node, bd.ACCEPTVoteproof(), ss); err != nil {
		return nil, err
	} else if j, err := cs.blockData.Writer().ReadACCEPTVoteproof(i); err != nil {
		return nil, err
	} else if j != nil {
		blk = blk.SetACCEPTVoteproof(j)
	}

	if i, err := cs.fetchBlockData(node, bd.SuffrageInfo(), ss); err != nil {
		return nil, err
	} else if j, err := cs.blockData.Writer().ReadSuffrageInfo(i); err != nil {
		return nil, err
	} else if j != nil {
		blk = blk.SetSuffrageInfo(j)
	}

	if i, err := cs.fetchBlockData(node, bd.Proposal(), ss); err != nil {
		return nil, err
	} else if j, err := cs.blockData.Writer().ReadProposal(i); err != nil {
		return nil, err
	} else if j != nil {
		blk = blk.SetProposal(j)
	}

	l.Debug().Hinted("block", blk.Hash()).Msg("fetched block")

	return blk, nil
}

func (cs *GeneralSyncer) commit() error {
	cs.Log().Debug().Msg("trying to commit")

	from := cs.heightFrom.Int64()
	to := cs.heightTo.Int64()

	if err := cs.database().Commit(); err != nil {
		return err
	}

	for i := from; i <= to; i++ {
		switch m, found, err := cs.database().Manifest(base.Height(i)); {
		case !found:
			return util.NotFoundError.Errorf("block, %v guessed to be stored, but not found", base.Height(i))
		case err != nil:
			return err
		default:
			cs.Log().Info().
				Hinted("proposal_hash", m.Proposal()).
				Dict("block", logging.Dict().
					Hinted("hash", m.Hash()).
					Hinted("height", m.Height()).
					Hinted("round", m.Round()),
				).
				Msg("new block stored")
		}
	}

	cs.Log().Debug().Msg("committed")

	return nil
}

func (cs *GeneralSyncer) rollback(rollbackCtx *BlockIntegrityError) error {
	cs.Log().Debug().Hinted("compare_from", rollbackCtx.From.Height()).Msg("block integrity failed; will rollback")

	var unmatched base.Height
	switch u, err := cs.compareBlocks(rollbackCtx.From.Height()); {
	case err != nil:
		return xerrors.Errorf("failed to check blocks: %w", err)
	case u <= base.NilHeight:
		return xerrors.Errorf("unmatched block not found; prepare() again")
	default:
		unmatched = u
	}

	cs.Log().Debug().Hinted("unmatched", unmatched).Msg("found unmatched; clean blocks")

	// NOTE clean block until unmatched height and start again prepare()
	var baseManifest block.Manifest
	if err := blockdata.CleanByHeight(cs.ost, cs.blockData, unmatched); err != nil {
		return err
	} else if unmatched > base.PreGenesisHeight+1 {
		switch m, found, err := cs.ost.ManifestByHeight(unmatched - 1); {
		case err != nil:
			return err
		case !found:
			return xerrors.Errorf("base manifest, %d for rollback not found", unmatched-1)
		default:
			baseManifest = m
		}
	}

	{
		cs.Lock()
		cs.heightFrom = unmatched
		cs.baseManifest = baseManifest
		cs.Unlock()
	}

	cs.Log().Debug().
		Hinted("new_height_from", unmatched).
		Msg("height from and base manifest was changed")

	return cs.prepare()
}

func (cs *GeneralSyncer) compareBlocks(from base.Height) (base.Height, error) {
	cs.Log().Debug().Hinted("compare_from", from).Msg("before rollback, check genesis blocks")

	cs.Log().Debug().Msg("compare genesis blocks")
	switch matched, err := cs.compareBlock(base.PreGenesisHeight + 1); {
	case err != nil:
		return base.NilHeight, xerrors.Errorf("failed to compare genesis block does not match: %w", err)
	case !matched:
		return base.PreGenesisHeight, nil // NOTE if genesis block does not match, will sync from PreGenesisHeight
	default:
		cs.Log().Debug().Msg("genesis blocks matched")
	}

	if from <= base.PreGenesisHeight+1 {
		cs.Log().Debug().Msg("all blocks matched")

		return base.NilHeight, nil
	}

	cs.Log().Debug().Int64("compare_from", from.Int64()).Msg("compare all inside blocks")
	switch unmatched, found, err := cs.compareInsideBlocks(from); {
	case err != nil:
		return base.NilHeight, err
	case found:
		return unmatched, nil
	}

	cs.Log().Debug().Msg("all blocks matched")

	return base.NilHeight, nil
}

func (cs *GeneralSyncer) compareInsideBlocks(top base.Height) (base.Height, bool, error) {
	switch unmatched, err := cs.searchUnmatched((base.PreGenesisHeight + 2), top); {
	case err != nil:
		return base.NilHeight, false, err
	case unmatched == base.NilHeight:
		return base.NilHeight, false, nil
	default:
		return unmatched, true, nil
	}
}

func (cs *GeneralSyncer) compareBlock(height base.Height) (bool, error) {
	var local block.Manifest
	switch m, found, err := cs.ost.ManifestByHeight(height); {
	case !found:
		return false, xerrors.Errorf("local block, %d not found", height)
	case err != nil:
		return false, xerrors.Errorf("failed to get local block, %d: %w", height, err)
	default:
		local = m
	}

	switch fetched, _, err := cs.fetchManifestsByNodes([]base.Height{height}); {
	case len(fetched) != 1:
		return false, xerrors.Errorf("empty manifest returned")
	case err != nil:
		return false, err
	default:
		return local.Hash().Equal(fetched[0].Hash()), nil
	}
}

func (cs *GeneralSyncer) searchUnmatched(from, to base.Height) (base.Height, error) {
	counted := int((to - from).Int64()) + 1

	var foundError error
	found := sort.Search(counted, func(i int) bool {
		if foundError != nil {
			return false
		}

		h := base.Height(from.Int64() + int64(i))
		if matched, err := cs.compareBlock(h); err != nil {
			foundError = err

			return false
		} else {
			return !matched
		}
	})

	if foundError != nil {
		return base.NilHeight, foundError
	} else if found == counted {
		return base.NilHeight, nil
	}

	return from + base.Height(int64(found)), nil
}

func (cs *GeneralSyncer) saveBlockData(sessions []blockdata.Session) ([]block.BlockDataMap, error) {
	cs.RLock()
	defer cs.RUnlock()

	maps := make([]block.BlockDataMap, len(sessions))
	for i := range sessions {
		ss := sessions[i]
		if ss == nil {
			return nil, xerrors.Errorf("empty block data session, %d found", i)
		}

		if j, err := cs.blockData.SaveSession(ss); err != nil {
			return nil, err
		} else {
			maps[i] = j
		}
	}

	return maps, nil
}

func (cs *GeneralSyncer) fetchBlockDataMaps(node network.Node, heights []base.Height) ([]block.BlockDataMap, error) {
	var maps []block.BlockDataMap
	switch i, err := node.Channel().BlockDataMaps(context.TODO(), heights); {
	case err != nil:
		return nil, err
	case len(i) != len(heights):
		return nil, xerrors.Errorf("failed to fetch block data map for manifests")
	default:
		maps = i
	}

	sort.SliceStable(maps, func(i, j int) bool {
		return maps[i].Height() < maps[j].Height()
	})

	for i := range heights {
		if maps[i].Height() != heights[i] {
			return nil, xerrors.Errorf("failed to fetch block data map for manifests; map has wrong height")
		}
	}

	return maps, nil
}

func (cs *GeneralSyncer) fetchBlockData(
	node network.Node,
	item block.BlockDataMapItem,
	ss blockdata.Session,
) (io.ReadSeeker, error) {
	var r io.ReadCloser
	if block.IsLocalBlockDateItem(item.URL()) {
		if i, err := node.Channel().BlockData(context.Background(), item); err != nil {
			return nil, err
		} else {
			r = i
		}
	} else if i, err := network.FetchBlockDataFromRemote(context.Background(), item); err != nil {
		return nil, err
	} else {
		r = i
	}

	defer func() {
		_ = r.Close()
	}()

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r); err != nil {
		return nil, err
	}

	s := bytes.NewReader(buf.Bytes())
	if _, err := ss.Import(item.Type(), s); err != nil {
		return nil, err
	}

	_, _ = s.Seek(0, 0)

	return s, nil
}

func (cs *GeneralSyncer) database() storage.SyncerSession {
	cs.RLock()
	defer cs.RUnlock()

	return cs.st
}

func (cs *GeneralSyncer) setState(state SyncerState, force bool) {
	cs.Lock()
	defer cs.Unlock()

	cs.Log().Debug().Str("new_state", state.String()).Bool("force", force).Msg("state changed")

	if !force && cs.state >= state {
		return
	}

	cs.state = state

	if cs.stateChan != nil && state != SyncerCreated {
		go func() {
			var blocks []block.Block
			if state == SyncerSaved {
				blocks = cs.blocks
			}

			cs.stateChan <- NewSyncerStateChangedContext(cs, state, blocks)
		}()
	}
}

func (cs *GeneralSyncer) initializeProvedNodes() {
	provedNodes := make([]base.Address, len(cs.sourceNodes))
	{
		var i int
		for k := range cs.sourceNodes {
			provedNodes[i] = k
			i++
		}
	}

	cs.pn = provedNodes
}

func (cs *GeneralSyncer) provedNodes() []base.Address {
	cs.RLock()
	defer cs.RUnlock()

	return cs.pn
}

func (cs *GeneralSyncer) setProvedNodes(pn []base.Address) {
	cs.Lock()
	defer cs.Unlock()

	cs.pn = pn
}

func validateSyncerSourceNodes(local network.Node, sourceNodes []network.Node) (map[base.Address]network.Node, error) {
	filtered := map[base.Address]network.Node{}
	for _, node := range sourceNodes {
		if local.Address().Equal(node.Address()) {
			return nil, xerrors.Errorf("one of sourceNodes is same with local node")
		}

		if _, found := filtered[node.Address()]; found {
			return nil, xerrors.Errorf("duplicated node found")
		}

		filtered[node.Address()] = node
	}

	return filtered, nil
}
