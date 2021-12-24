package isaac

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

var blockIntegrityError = util.NewError("block integrity failed")

type BlockIntegrityError struct {
	*util.NError
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
	odatabase               storage.Database
	blockdata               blockdata.Blockdata
	policy                  *LocalPolicy
	stLock                  sync.RWMutex
	st                      storage.SyncerSession
	sourceChannelsFunc      func() map[string]network.Channel
	heightFrom              base.Height
	heightTo                base.Height
	limitManifestsPerWorker int
	limitBlocksPerOnce      int
	pchs                    *util.LockedItem
	state                   SyncerState
	baseManifest            block.Manifest
	stateChan               chan<- SyncerStateChangedContext
	tailManifest            block.Manifest
	blksLock                sync.RWMutex
	blks                    []block.Block
	blockdataSessions       []blockdata.Session
	lifeCtx                 context.Context
	lifeCancel              func()
	donechLock              sync.RWMutex
	donech                  chan bool
}

func NewGeneralSyncer(
	odb storage.Database,
	bd blockdata.Blockdata,
	policy *LocalPolicy,
	sourceChannelsFunc func() map[string]network.Channel,
	baseManifest block.Manifest,
	to base.Height,
) (*GeneralSyncer, error) {
	var from base.Height
	if baseManifest == nil {
		from = base.PreGenesisHeight
	} else {
		from = baseManifest.Height() + 1
	}

	if from > to {
		return nil, errors.Errorf("from height, %d is greater than to height, %d", from, to)
	}

	if m, found, err := odb.LastManifest(); err != nil {
		return nil, err
	} else if found && from <= m.Height() {
		return nil, errors.Errorf("from height is same or lower than last block; from=%d last=%d", from, m.Height())
	}

	lifeCtx, lifeCancel := context.WithCancel(context.Background())

	return &GeneralSyncer{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.
				Int64("from", from.Int64()).Int64("to", to.Int64()).
				Stringer("syncer_id", util.UUID()).
				Str("module", "general-syncer")
		}),
		odatabase:               odb,
		blockdata:               bd,
		policy:                  policy,
		sourceChannelsFunc:      sourceChannelsFunc,
		heightFrom:              from,
		heightTo:                to,
		baseManifest:            baseManifest,
		limitManifestsPerWorker: 10,
		limitBlocksPerOnce:      10,
		pchs:                    util.NewLockedItem(nil),
		state:                   SyncerCreated,
		blks:                    make([]block.Block, to-from+1),
		blockdataSessions:       make([]blockdata.Session, to-from+1),
		lifeCtx:                 lifeCtx,
		lifeCancel:              lifeCancel,
	}, nil
}

func (cs *GeneralSyncer) SetLogging(l *logging.Logging) *logging.Logging {
	return cs.Logging.SetLogging(l)
}

func (cs *GeneralSyncer) ID() string {
	return fmt.Sprintf("%v-%v", cs.heightFrom, cs.heightTo)
}

func (cs *GeneralSyncer) Close() error {
	st := cs.syncerSession()
	if st == nil {
		return nil
	}

	if cs.lifeCancel != nil {
		cs.lifeCancel()
	}

	cs.donechLock.RLock()
	if cs.donech != nil {
		<-cs.donech
	}
	cs.donechLock.RUnlock()

	defer cs.Log().Debug().Msg("closed")
	defer cs.setSyncerSession(nil)

	return st.Close()
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
	if cs.State() >= SyncerPrepared {
		cs.Log().Debug().Msg("already prepared")

		return nil
	}

	donech := make(chan bool, 2)
	cs.donechLock.Lock()
	cs.donech = donech
	cs.donechLock.Unlock()

	go func() {
		// NOTE do forever unless successfully done
		defer func() {
			donech <- true
		}()

		for {
			select {
			case <-cs.lifeCtx.Done():
				return
			default:
				err := cs.prepare()
				if err == nil {
					return
				}

				if errors.Is(err, util.IgnoreError) {
					return
				}

				cs.Log().Error().Err(err).Msg("failed to prepare for syncing")

				var rollbackCtx *BlockIntegrityError
				if errors.As(err, &rollbackCtx) {
					if e := cs.rollback(rollbackCtx); e == nil {
						return
					}

					cs.Log().Error().Err(err).Msg("failed to rollback")

					<-time.After(time.Millisecond * 500)

					continue
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

	if cs.State() < SyncerPrepared {
		select {
		case <-cs.lifeCtx.Done():
			return util.IgnoreError.Errorf("stopped")
		default:
			if err := cs.headAndTailManifests(); err != nil {
				return err
			}
		}

		select {
		case <-cs.lifeCtx.Done():
			return util.IgnoreError.Errorf("stopped")
		default:
			if err := cs.fillManifests(); err != nil {
				return err
			}
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
		if errors.Is(err, util.IgnoreError) {
			return nil
		}

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

	st := cs.syncerSession()
	if st != nil {
		cs.Log().Debug().Int("blocks", len(cs.blocks())).Msg("reset: will cleanup storage; database and block data")
		if err := blockdata.CleanByHeight(cs.odatabase, cs.blockdata, cs.heightFrom); err != nil {
			return err
		}

		// NOTE clean up blocks in block data session
		cs.Log().Debug().Int("blocks", len(cs.blocks())).Msg("reset: will cleanup block data session")
		for i := range cs.blockdataSessions {
			ss := cs.blockdataSessions[i]
			if ss == nil {
				continue
			}

			if err := ss.Cancel(); err != nil {
				return err
			}
		}

		if err := st.Close(); err != nil {
			return err
		}
	}

	if err := cs.resetProvedChannels(); err != nil {
		return err
	}

	cs.blockdataSessions = make([]blockdata.Session, cs.heightTo-cs.heightFrom+1)

	i, err := cs.odatabase.NewSyncerSession()
	if err != nil {
		return err
	}

	if sl, ok := i.(logging.SetLogging); ok {
		_ = sl.SetLogging(cs.Logging)
	}

	cs.setSyncerSession(i)

	return nil
}

func (cs *GeneralSyncer) headAndTailManifests() error {
	if cs.State() != SyncerPreparing {
		cs.Log().Debug().Stringer("state", cs.State()).Msg("not preparing state")

		return nil
	}

	var heights []base.Height
	if cs.heightFrom == cs.heightTo {
		heights = []base.Height{cs.heightFrom}
	} else {
		heights = []base.Height{cs.heightFrom, cs.heightTo}
	}

	var manifests []block.Manifest
	var chs []string
	switch ms, p, err := cs.fetchManifestsByChannels(heights); {
	case err != nil:
		return err
	case len(ms) < 1:
		return errors.Errorf("failed to fetch manifests from all of source")
	default:
		manifests = ms
		chs = p
	}

	if cs.baseManifest != nil {
		head := manifests[0]
		cs.Log().Debug().
			Stringer("base_manifest_previous", cs.baseManifest.PreviousBlock()).
			Stringer("base_manifest", cs.baseManifest.Hash()).
			Stringer("head_previous", head.PreviousBlock()).
			Stringer("head", head.Hash()).
			Msg("checking base and head manifest")

		checker := NewManifestsValidationChecker(cs.policy.NetworkID(), []block.Manifest{cs.baseManifest, head})
		_ = checker.SetLogging(cs.Logging)

		if err := util.NewChecker("sync-manifests-validation-checker", []util.CheckerFunc{
			checker.CheckSerialized,
		}).Check(); err != nil {
			cs.Log().Error().Err(err).Msg("failed to verify manifests")
			return err
		}
	}

	cs.setProvedChannels(chs)

	st := cs.syncerSession()
	if err := st.SetManifests(manifests); err != nil {
		return err
	}

	cs.setTailManifest(manifests[len(manifests)-1])

	return nil
}

func (cs *GeneralSyncer) fillManifests() error {
	if cs.State() != SyncerPreparing {
		cs.Log().Debug().Stringer("state", cs.State()).Msg("not preparing state")

		return nil
	}

	if cs.heightFrom == cs.heightTo || cs.heightTo == cs.heightFrom+1 {
		return nil
	}

	fill := func(heights []base.Height) error {
		switch ms, p, err := cs.fetchManifestsByChannels(heights); {
		case err != nil:
			return err
		case len(ms) < 1:
			return errors.Errorf("failed to fetch manifests from all of source")
		case len(p) < 1:
			return errors.Errorf("empty proved channels")
		default:
			cs.setProvedChannels(p)

			return cs.syncerSession().SetManifests(ms)
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
		}
		heights = nil
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
		return errors.Errorf("not saving state: %v", cs.State())
	}

	cs.Log().Debug().Msg("start to fetch blocks")
	defer cs.Log().Debug().Msg("fetched blocks")

	for {
		select {
		case <-cs.lifeCtx.Done():
			return util.IgnoreError.Errorf("stopped")
		default:
			if err := cs.fetchBlocksByChannels(); err != nil {
				cs.Log().Error().Err(err).Msg("failed to fetch blocks by channels")

				<-time.After(time.Millisecond * 500)

				continue
			}

			return nil
		}
	}
}

func (cs *GeneralSyncer) fetchBlocksByChannels() error {
	cs.Log().Debug().Msg("start to fetch blocks by channels")

	worker := util.NewParallelWorker("sync-fetch-blocks", 5)
	defer worker.Done()
	_ = worker.SetLogging(cs.Logging)

	chs := cs.provedChannels()

	if len(chs) < 1 {
		return errors.Errorf("empty proved channels")
	}

	for i := range chs {
		worker.Run(cs.workerCallbackFetchBlocks(i, chs[i]))
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

	cs.Log().Debug().Msg("fetched blocks by channels")

	// check fetched blocks
	for i := cs.heightFrom; i <= cs.heightTo; i++ {
		if found, err := cs.syncerSession().HasBlock(i); err != nil {
			return errors.Wrapf(err, "some block not found after fetching blocks: height=%d", i)
		} else if !found {
			return errors.Errorf("some block not found after fetching blocks: height=%d", i)
		}
	}

	return nil
}

func (cs *GeneralSyncer) handleSyncerFetchBlockError(err error) error {
	if err == nil {
		return nil
	}

	var fm *syncerFetchBlockError
	if !errors.As(err, &fm) {
		cs.Log().Error().Err(err).Msg("something wrong to fetch blocks")
		return nil
	}

	if fm.err != nil {
		cs.Log().Error().Err(err).
			Str("source", fm.source).Msg("something wrong to fetch blocks from channel")

		return errors.Wrap(fm.err, "failed to fetch blocks")
	}

	if len(fm.blocks) < 1 {
		cs.Log().Error().Err(err).Str("source", fm.source).
			Msg("empty blocks; something wrong to fetch blocks from channel")

		return errors.Errorf("empty blocks; failed to fetch blocks")
	}

	if ms, err := cs.checkFetchedBlocks(fm.blocks); err != nil {
		return err
	} else if len(fm.missing) > 0 || len(ms) > 0 {
		cs.Log().Error().Interface("missing_blocks", len(fm.missing)+len(ms)).Msg("still missing blocks found")

		return errors.Errorf("some missing blocks found; failed to fetch blocks")
	}

	return nil
}

func (cs *GeneralSyncer) distributeBlocksJob(worker *util.ParallelWorker) error {
	from := cs.heightFrom.Int64()
	to := cs.heightTo.Int64()

	limit := cs.limitBlocksPerOnce
	{ // more widely distribute requests
		total := int(to - from)
		if total < len(cs.provedChannels())*limit {
			limit = total / len(cs.provedChannels())
		}
	}

	var heights []base.Height
	for i := from; i <= to; i++ {
		if found, err := cs.syncerSession().HasBlock(base.Height(i)); err != nil {
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

func (cs *GeneralSyncer) fetchManifestsByChannels(heights []base.Height) ([]block.Manifest, []string, error) {
	l := cs.Log().With().Int64("height_from", heights[0].Int64()).
		Int64("height_to", heights[len(heights)-1].Int64()).
		Logger()

	l.Debug().Msg("trying to fetch manifest")

	chs := cs.provedChannels()

	wk := util.NewDistributeWorker(cs.lifeCtx, int64(len(chs)), nil)
	defer wk.Close()

	resultch := make(chan map[string][]block.Manifest)
	donech := make(chan struct{})
	fetched := map[string][]block.Manifest{}
	go func() {
		for result := range resultch {
			for source, manifests := range result {
				fetched[source] = manifests
			}
		}

		donech <- struct{}{}
	}()

	go func() {
		for i := range chs {
			source := i
			ch := chs[i]

			if err := wk.NewJob(func(context.Context, uint64) error {
				i, err := cs.callbackFetchManifests(source, ch, heights)
				if err != nil {
					l.Error().Err(err).Str("source", source).Msg("failed to get manifest from channel")

					return nil
				}

				resultch <- map[string][]block.Manifest{source: i}

				return nil
			}); err != nil {
				break
			}
		}

		wk.Done()
	}()

	if err := wk.Wait(); err != nil {
		return nil, nil, err
	}

	close(resultch)

	<-donech

	cs.Log().Debug().Int("fetched", len(fetched)).Msg("fetched manifests")

	switch ms, p, err := cs.checkThresholdByHeights(heights, fetched); {
	case err != nil:
		return nil, nil, err
	case len(p) < 1:
		return nil, nil, errors.Errorf("empty proved channels")
	default:
		for i, height := range heights {
			b := ms[i]
			if height != b.Height() {
				return nil, nil, errors.Errorf("invalid Manifest found; height does not match")
			}
		}

		return ms, p, nil
	}
}

func (cs *GeneralSyncer) callbackFetchManifests(
	source string,
	ch network.Channel,
	heights []base.Height,
) ([]block.Manifest, error) {
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

		i, err := cs.callbackFetchManifestsSlice(source, ch, sliced)
		if err != nil {
			return nil, err
		}
		update(i)
		sliced = nil
	}

	if len(sliced) > 0 {
		i, err := cs.callbackFetchManifestsSlice(source, ch, sliced)
		if err != nil {
			return nil, err
		}
		update(i)
	}

	return manifests, nil
}

func (cs *GeneralSyncer) callbackFetchManifestsSlice(
	source string, ch network.Channel, heights []base.Height,
) ([]block.Manifest, error) {
	var maxRetries uint = 3

	l := cs.Log().With().Uint("max-retries", maxRetries).
		Str("source", source).
		Interface("heights", heights).
		Logger()

	l.Debug().Msg("trying to fetch manifest of channel")

	var manifests []block.Manifest

	missing := heights

	if err := util.Retry(maxRetries, time.Millisecond*300, func(retries int) error {
		l.Debug().Int("retries", retries).Msg("try to fetch manifest")

		bs, err := cs.fetchManifests(ch, missing)
		switch {
		case errors.Is(err, context.Canceled):
			return util.StopRetryingError.Wrap(err)
		case err != nil:
			return err
		}

		ss, ms, err := cs.sanitizeManifests(heights, bs)
		if err != nil {
			return err
		}
		manifests = ss
		missing = ms

		if len(missing) > 0 {
			return errors.Errorf("something missing")
		}

		return nil
	}); err != nil {
		return nil, err
	}

	l.Debug().Interface("missing", missing).Int("fetched", len(manifests)).Msg("fetched manifest of channel")

	return manifests, nil
}

func (cs *GeneralSyncer) checkThresholdByHeights(heights []base.Height, fetched map[string][]block.Manifest) (
	[]block.Manifest, // major manifests
	[]string, // sources
	error,
) {
	manifests := make([]block.Manifest, len(heights))

	var ps []string
	for index := range heights {
		m, p, err := cs.checkThreshold(index, heights, fetched)
		if err != nil {
			return nil, nil, err
		}
		manifests[index] = m
		ps = p
	}

	return manifests, ps, nil
}

func (cs *GeneralSyncer) checkThreshold(
	index int,
	heights []base.Height,
	fetched map[string][]block.Manifest,
) (block.Manifest, []string, error) {
	height := heights[index]
	hashByChannel := map[string][]string{}
	ms := map[string]block.Manifest{}

	var set []string // nolint
	for no := range fetched {
		bs := fetched[no]
		if len(bs) != len(heights) {
			cs.Log().Debug().Int("expected", len(heights)).Int("returned", len(bs)).
				Msg("failed to get the expected data from channel")

			continue
		}

		if bs[index] == nil {
			continue
		}

		key := bs[index].Hash().String()
		set = append(set, key)
		ms[key] = bs[index]
		hashByChannel[key] = append(hashByChannel[key], no)
	}

	if len(set) < 1 {
		return nil, nil, errors.Errorf("nothing fetched for height=%d", height)
	}

	threshold, err := base.NewThreshold(uint(len(set)), cs.policy.ThresholdRatio())
	if err != nil {
		return nil, nil, err
	}

	result, key := base.FindMajorityFromSlice(threshold.Total, threshold.Threshold, set)
	if result != base.VoteResultMajority {
		return nil, nil, errors.Errorf("given target channel doet not have common blocks: height=%s", height)
	}

	return ms[key], hashByChannel[key], nil
}

func (cs *GeneralSyncer) fetchManifests(ch network.Channel, heights []base.Height) ([]block.Manifest, error) { // nolint
	maps, err := cs.fetchBlockdataMaps(ch, heights)
	if err != nil {
		return nil, err
	}

	wk := util.NewErrgroupWorker(cs.lifeCtx, int64(len(heights)))
	defer wk.Close()

	resultch := make(chan block.Manifest)
	donech := make(chan struct{})
	var fetched []block.Manifest
	go func() {
		for i := range resultch {
			fetched = append(fetched, i)
		}

		donech <- struct{}{}
	}()

	go func() {
		for i := range maps {
			bd := maps[i]
			if err := wk.NewJob(func(ctx context.Context, _ uint64) error {
				r, err := ch.Blockdata(ctx, bd.Manifest())
				if err != nil {
					return err
				}

				defer func() {
					_ = r.Close()
				}()

				m, err := cs.blockdata.Writer().ReadManifest(r)
				if err != nil {
					return err
				}

				if err := block.CompareManifestWithMap(m, bd); err != nil {
					return err
				}

				resultch <- m

				return nil
			}); err != nil {
				break
			}
		}

		wk.Done()
	}()

	if err := wk.Wait(); err != nil {
		return nil, err
	}
	close(resultch)

	<-donech

	return fetched, nil
}

// sanitizeManifests checks and filter the fetched Manifests. NOTE the
// input heights should be sorted by it's Height.
func (*GeneralSyncer) sanitizeManifests(heights []base.Height, l interface{}) (
	[]block.Manifest, []base.Height, error,
) {
	var checked []block.Manifest
	var missing []base.Height
	if len(heights) < 1 {
		return checked, missing, nil
	}

	var bs []block.Manifest
	switch t := l.(type) {
	case []block.Block:
		for i := range t {
			bs = append(bs, t[i])
		}
	case []block.Manifest:
		bs = t
	default:
		return nil, nil, errors.Errorf("not Manifest like: %T", l)
	}

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

func (cs *GeneralSyncer) workerCallbackFetchBlocks(source string, ch network.Channel) util.WorkerCallback {
	return func(jobID uint, job interface{}) error {
		heights, ok := job.([]base.Height)
		if !ok {
			return errors.Errorf("job is not []Height: %T", job)
		}

		l := cs.Log().With().Str("source", source).
			Interface("heights", heights).
			Logger()

		var manifests []block.Manifest
		var missing []base.Height
		var err error
		if bs, e := cs.fetchBlocks(source, ch, heights); err != nil {
			err = e
		} else if manifests, missing, err = cs.sanitizeManifests(heights, bs); err != nil {
			err = e
		}

		if err != nil {
			l.Error().Err(err).Msg("failed to fetch blocks")

			return &syncerFetchBlockError{
				source:  source,
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
			source:  source,
			heights: heights,
			err:     err,
			blocks:  blocks,
			missing: missing,
		}
	}
}

func (cs *GeneralSyncer) checkFetchedBlocks(fetched []block.Block) ([]base.Height, error) {
	networkID := cs.policy.NetworkID()

	var filtered []block.Block       // nolint
	var sessions []blockdata.Session // nolint
	var missing []base.Height
	for i := range fetched {
		blk := fetched[i]
		if err := blk.IsValid(networkID); err != nil {
			cs.Log().Error().Err(err).
				Int64("height", blk.Height().Int64()).
				Object("block", blk).
				Msg("found invalid block")

			missing = append(missing, blk.Height())

			continue
		}

		switch manifest, found, err := cs.syncerSession().Manifest(blk.Height()); {
		case err != nil:
			return nil, err
		case !found:
			return nil, util.NotFoundError.Errorf("manifest not found")
		case !manifest.Hash().Equal(blk.Hash()):
			missing = append(missing, blk.Height())

			continue
		}

		ss := cs.blockdataSessions[blk.Height()-cs.heightFrom]
		if ss == nil {
			missing = append(missing, blk.Height())

			continue
		}

		sessions = append(sessions, ss)
		filtered = append(filtered, blk)
	}

	if len(missing) > 0 {
		return missing, nil
	}

	cs.setBlocks(filtered)

	if maps, err := cs.saveBlockdata(sessions); err != nil {
		return nil, err
	} else if err := cs.syncerSession().SetBlocks(filtered, maps); err != nil {
		return nil, err
	} else {
		return nil, nil
	}
}

func (cs *GeneralSyncer) fetchBlocks(
	source string,
	ch network.Channel,
	heights []base.Height,
) ([]block.Block, error) {
	l := cs.Log().With().Str("source", source).
		Int64("height_from", heights[0].Int64()).
		Int64("height_to", heights[len(heights)-1].Int64()).
		Logger()

	maps, err := cs.fetchBlockdataMaps(ch, heights)
	if err != nil {
		return nil, err
	}

	l.Debug().Msg("trying to fetch blocks")

	fetched := make([]block.Block, len(heights))
	for i := range maps {
		j, err := cs.fetchBlock(source, ch, maps[i])
		switch {
		case err != nil:
			l.Error().Err(err).Msg("failed to fetch block")

			return nil, err
		case j == nil:
			err = errors.Errorf("empty block found")
			l.Error().Err(err).Msg("failed to fetch block")

			return nil, err
		default:
			fetched[i] = j
		}
	}

	sort.SliceStable(fetched, func(i, j int) bool {
		return fetched[i].Height() < fetched[j].Height()
	})

	l.Debug().Int("fetched", len(fetched)).Msg("fetched blocks")

	return fetched, nil
}

func (cs *GeneralSyncer) blockdataSession(height base.Height) (blockdata.Session, error) {
	cs.Lock()
	defer cs.Unlock()

	if ss := cs.blockdataSessions[height-cs.heightFrom]; ss != nil {
		return ss, nil
	}

	i, err := cs.blockdata.NewSession(height)
	if err != nil {
		return nil, err
	}
	cs.blockdataSessions[height-cs.heightFrom] = i

	return i, nil
}

func (cs *GeneralSyncer) fetchBlock( // revive:disable-line:cognitive-complexity,cyclomatic,line-length-limit
	source string,
	ch network.Channel,
	bd block.BlockdataMap,
) (block.Block, error) {
	ss, err := cs.blockdataSession(bd.Height())
	if err != nil {
		return nil, err
	}

	l := cs.Log().With().Str("source", source).
		Int64("height", bd.Height().Int64()).
		Logger()

	l.Debug().Msg("trying to fetch block")

	blk := (interface{})(block.EmptyBlockV0()).(block.BlockUpdater)

	switch i, found, err := cs.syncerSession().Manifest(bd.Height()); {
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

	if i, err := cs.fetchBlockdata(ch, bd.Operations(), ss); err != nil {
		return nil, err
	} else if j, err := cs.blockdata.Writer().ReadOperations(i); err != nil {
		return nil, err
	} else if j != nil {
		blk = blk.SetOperations(j)
	}

	if i, err := cs.fetchBlockdata(ch, bd.OperationsTree(), ss); err != nil {
		return nil, err
	} else if j, err := cs.blockdata.Writer().ReadOperationsTree(i); err != nil {
		return nil, err
	} else if j.Len() > 0 {
		blk = blk.SetOperationsTree(j)
	}

	if i, err := cs.fetchBlockdata(ch, bd.States(), ss); err != nil {
		return nil, err
	} else if j, err := cs.blockdata.Writer().ReadStates(i); err != nil {
		return nil, err
	} else if j != nil {
		blk = blk.SetStates(j)
	}

	if i, err := cs.fetchBlockdata(ch, bd.StatesTree(), ss); err != nil {
		return nil, err
	} else if j, err := cs.blockdata.Writer().ReadStatesTree(i); err != nil {
		return nil, err
	} else if j.Len() > 0 {
		blk = blk.SetStatesTree(j)
	}

	if i, err := cs.fetchBlockdata(ch, bd.INITVoteproof(), ss); err != nil {
		return nil, err
	} else if j, err := cs.blockdata.Writer().ReadINITVoteproof(i); err != nil {
		return nil, err
	} else if j != nil {
		blk = blk.SetINITVoteproof(j)
	}

	if i, err := cs.fetchBlockdata(ch, bd.ACCEPTVoteproof(), ss); err != nil {
		return nil, err
	} else if j, err := cs.blockdata.Writer().ReadACCEPTVoteproof(i); err != nil {
		return nil, err
	} else if j != nil {
		blk = blk.SetACCEPTVoteproof(j)
	}

	if i, err := cs.fetchBlockdata(ch, bd.SuffrageInfo(), ss); err != nil {
		return nil, err
	} else if j, err := cs.blockdata.Writer().ReadSuffrageInfo(i); err != nil {
		return nil, err
	} else if j != nil {
		blk = blk.SetSuffrageInfo(j)
	}

	if i, err := cs.fetchBlockdata(ch, bd.Proposal(), ss); err != nil {
		return nil, err
	} else if j, err := cs.blockdata.Writer().ReadProposal(i); err != nil {
		return nil, err
	} else if j != nil {
		blk = blk.SetProposal(j)
	}

	l.Debug().Stringer("block", blk.Hash()).Msg("fetched block")

	return blk, nil
}

func (cs *GeneralSyncer) commit() error {
	cs.Log().Debug().Msg("trying to commit")

	from := cs.heightFrom.Int64()
	to := cs.heightTo.Int64()

	if err := cs.syncerSession().Commit(); err != nil {
		return err
	}

	for i := from; i <= to; i++ {
		switch m, found, err := cs.syncerSession().Manifest(base.Height(i)); {
		case err != nil:
			return err
		case !found:
			return util.NotFoundError.Errorf("block, %v guessed to be stored, but not found", base.Height(i))
		default:
			cs.Log().Info().
				Stringer("proposal_hash", m.Proposal()).
				Object("block", m).
				Msg("new block stored")
		}
	}

	cs.Log().Debug().Msg("committed")

	return nil
}

func (cs *GeneralSyncer) rollback(rollbackCtx *BlockIntegrityError) error {
	cs.Log().Debug().Int64("compare_from", rollbackCtx.From.Height().Int64()).Msg("block integrity failed; will rollback")

	var unmatched base.Height
	switch u, err := cs.compareBlocks(rollbackCtx.From.Height()); {
	case err != nil:
		return errors.Wrap(err, "failed to check blocks")
	case u <= base.NilHeight:
		return errors.Errorf("unmatched block not found; prepare() again")
	default:
		unmatched = u
	}

	cs.Log().Debug().Int64("unmatched", unmatched.Int64()).Msg("found unmatched; clean blocks")

	// NOTE clean block until unmatched height and start again prepare()
	var baseManifest block.Manifest
	if err := blockdata.CleanByHeight(cs.odatabase, cs.blockdata, unmatched); err != nil {
		return err
	} else if unmatched > base.PreGenesisHeight+1 {
		switch m, found, err := cs.odatabase.ManifestByHeight(unmatched - 1); {
		case err != nil:
			return err
		case !found:
			return errors.Errorf("base manifest, %d for rollback not found", unmatched-1)
		default:
			baseManifest = m
		}
	}

	{
		cs.Lock()
		cs.heightFrom = unmatched
		cs.initBlocks()
		cs.baseManifest = baseManifest
		cs.Unlock()
	}

	cs.Log().Debug().
		Int64("new_height_from", unmatched.Int64()).
		Msg("height from and base manifest was changed")

	return cs.prepare()
}

func (cs *GeneralSyncer) compareBlocks(from base.Height) (base.Height, error) {
	cs.Log().Debug().Int64("compare_from", from.Int64()).Msg("before rollback, check genesis blocks")

	cs.Log().Debug().Msg("compare genesis blocks")
	switch matched, err := cs.compareBlock(base.PreGenesisHeight + 1); {
	case err != nil:
		return base.NilHeight, errors.Wrap(err, "failed to compare genesis block does not match")
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
	switch m, found, err := cs.odatabase.ManifestByHeight(height); {
	case err != nil:
		return false, errors.Wrapf(err, "failed to get local block, %d", height)
	case !found:
		return false, errors.Errorf("local block, %d not found", height)
	default:
		local = m
	}

	switch fetched, _, err := cs.fetchManifestsByChannels([]base.Height{height}); {
	case err != nil:
		return false, err
	case len(fetched) != 1:
		return false, errors.Errorf("empty manifest returned")
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
		matched, err := cs.compareBlock(h)
		if err != nil {
			foundError = err

			return false
		}
		return !matched
	})

	if foundError != nil {
		return base.NilHeight, foundError
	} else if found == counted {
		return base.NilHeight, nil
	}

	return from + base.Height(int64(found)), nil
}

func (cs *GeneralSyncer) saveBlockdata(sessions []blockdata.Session) ([]block.BlockdataMap, error) {
	cs.RLock()
	defer cs.RUnlock()

	maps := make([]block.BlockdataMap, len(sessions))
	for i := range sessions {
		ss := sessions[i]
		if ss == nil {
			return nil, errors.Errorf("empty block data session, %d found", i)
		}

		j, err := cs.blockdata.SaveSession(ss)
		if err != nil {
			return nil, err
		}
		maps[i] = j
	}

	return maps, nil
}

func (cs *GeneralSyncer) fetchBlockdataMaps(ch network.Channel, heights []base.Height) ([]block.BlockdataMap, error) {
	var maps []block.BlockdataMap
	switch i, err := ch.BlockdataMaps(cs.lifeCtx, heights); {
	case err != nil:
		return nil, err
	case len(i) != len(heights):
		return nil, errors.Errorf("failed to fetch block data map for manifests")
	default:
		maps = i
	}

	sort.SliceStable(maps, func(i, j int) bool {
		return maps[i].Height() < maps[j].Height()
	})

	for i := range heights {
		if maps[i].Height() != heights[i] {
			return nil, errors.Errorf("failed to fetch block data map for manifests; map has wrong height")
		}
	}

	return maps, nil
}

func (cs *GeneralSyncer) fetchBlockdata(
	ch network.Channel,
	item block.BlockdataMapItem,
	ss blockdata.Session,
) (io.ReadSeeker, error) {
	var r io.ReadCloser
	if block.IsLocalBlockdataItem(item.URL()) {
		i, err := ch.Blockdata(cs.lifeCtx, item)
		if err != nil {
			return nil, err
		}
		r = i
	} else if i, err := network.FetchBlockdataFromRemote(cs.lifeCtx, item); err != nil {
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

func (cs *GeneralSyncer) setState(state SyncerState, force bool) {
	cs.Lock()
	defer cs.Unlock()

	cs.Log().Debug().Stringer("new_state", state).Bool("force", force).Msg("state changed")

	if !force && cs.state >= state {
		return
	}

	cs.state = state

	if cs.stateChan != nil && state != SyncerCreated {
		var blocks []block.Block
		if state == SyncerSaved {
			blocks = make([]block.Block, len(cs.blocks()))
			copy(blocks, cs.blocks())
		}

		go func() {
			cs.stateChan <- NewSyncerStateChangedContext(cs, state, blocks)
		}()
	}
}

func (cs *GeneralSyncer) resetProvedChannels() error {
	chs := cs.sourceChannelsFunc()

	if len(chs) < 1 {
		return errors.Errorf("empty source channels")
	}

	cs.pchs.Set(chs)

	return nil
}

func (cs *GeneralSyncer) provedChannels() map[string]network.Channel {
	i := cs.pchs.Value()

	if i == nil {
		return nil
	}

	return i.(map[string]network.Channel)
}

func (cs *GeneralSyncer) setProvedChannels(p []string) {
	pchs := cs.provedChannels()
	chs := map[string]network.Channel{}
	for i := range p {
		ch, found := pchs[p[i]]
		if !found {
			continue
		}

		chs[p[i]] = ch
	}

	cs.pchs.Set(chs)
}

func (cs *GeneralSyncer) syncerSession() storage.SyncerSession {
	cs.stLock.RLock()
	defer cs.stLock.RUnlock()

	return cs.st
}

func (cs *GeneralSyncer) setSyncerSession(st storage.SyncerSession) {
	cs.stLock.Lock()
	defer cs.stLock.Unlock()

	cs.st = st
}

func (cs *GeneralSyncer) blocks() []block.Block {
	cs.blksLock.RLock()
	defer cs.blksLock.RUnlock()

	return cs.blks
}

func (cs *GeneralSyncer) initBlocks() {
	cs.blksLock.Lock()
	defer cs.blksLock.Unlock()

	cs.blks = make([]block.Block, cs.heightTo-cs.heightFrom+1)
}

func (cs *GeneralSyncer) setBlocks(blks []block.Block) {
	cs.blksLock.Lock()
	defer cs.blksLock.Unlock()

	for i := range blks {
		blk := blks[i]
		cs.blks[blk.Height()-cs.heightFrom] = blk
	}
}
