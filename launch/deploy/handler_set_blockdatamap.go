package deploy

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"golang.org/x/xerrors"
)

var LimitBlockDataMaps = 100

func NewSetBlockDataMapsHandler(
	enc encoder.Encoder,
	db storage.Database,
	bc *BlockDataCleaner,
) network.HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var bdms []block.BlockDataMap
		switch i, err := loadBlockDataMaps(r, enc); {
		case err != nil:
			network.WriteProblemWithError(w, http.StatusBadRequest, xerrors.Errorf("failed to load blockdatamaps: %w", err))
			return
		case len(i) < 1, len(i) > LimitBlockDataMaps:
			network.WriteProblemWithError(w, http.StatusBadRequest, err)
			return
		default:
			bdms = i
		}

		if err := checkBlockDataMaps(db, bdms); err != nil {
			network.WriteProblemWithError(w, http.StatusBadRequest, err)
			return
		}

		if err := commitBlockDataMaps(db, bc, bdms); err != nil {
			network.WriteProblemWithError(w, http.StatusInternalServerError, err)
			return
		}

		heights := make([]base.Height, len(bdms))

		var i int
		for j := range bdms {
			heights[i] = bdms[j].Height()
			i++
		}

		if i, err := jsonenc.Marshal(map[string]interface{}{
			"heights":       heights,
			"expected_time": localtime.UTCNow().Add(bc.RemoveAfter()),
		}); err != nil {
			network.WriteProblemWithError(w, http.StatusInternalServerError, err)
			return
		} else {
			w.Header().Set("Content-Type", "application/json")

			_, _ = w.Write(i)
		}
	}
}

func loadBlockDataMaps(r *http.Request, enc encoder.Encoder) ([]block.BlockDataMap, error) {
	var bs [][]byte
	if i, err := ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	} else if j, err := enc.UnmarshalArray(i); err != nil {
		return nil, err
	} else {
		bs = j
	}

	founds := map[base.Height]bool{}
	ubd := make([]block.BlockDataMap, len(bs))
	for i := range bs {
		if j, err := enc.DecodeByHint(bs[i]); err != nil {
			return nil, err
		} else if k, ok := j.(block.BlockDataMap); !ok {
			return nil, xerrors.Errorf("not block.BlockDataMap type, %T", j)
		} else if _, found := founds[k.Height()]; found {
			continue
		} else {
			ubd[i] = k
			founds[k.Height()] = true
		}
	}

	return ubd, nil
}

func checkBlockDataMaps(db storage.Database, bdms []block.BlockDataMap) error {
	var limit int64 = 100
	sem := semaphore.NewWeighted(limit)
	eg, ctx := errgroup.WithContext(context.Background())

	for i := range bdms {
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		bdm := bdms[i]
		eg.Go(func() error {
			defer sem.Release(1)

			return checkBlockDataMap(db, bdm)
		})
	}

	if err := sem.Acquire(ctx, limit); err != nil {
		if !xerrors.Is(err, context.Canceled) {
			return err
		}
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func checkBlockDataMap(db storage.Database, bdm block.BlockDataMap) error {
	if err := bdm.IsValid(nil); err != nil {
		return err
	}

	switch i, found, err := db.ManifestByHeight(bdm.Height()); {
	case err != nil:
		return err
	case !found:
		return util.NotFoundError.Errorf("height of blockdatamap, %d not found in database", bdm.Height())
	case !i.Hash().Equal(bdm.Block()):
		return xerrors.Errorf("block hash does not match with manifest")
	}

	return nil
}

func commitBlockDataMaps(db storage.Database, bc *BlockDataCleaner, bdms []block.BlockDataMap) error {
	if err := db.SetBlockDataMaps(bdms); err != nil {
		return err
	}

	var limit int64 = 100
	sem := semaphore.NewWeighted(limit)
	eg, ctx := errgroup.WithContext(context.Background())

	for i := range bdms {
		bdm := bdms[i]
		if bdm.IsLocal() { // NOTE local blockdata will not be removed
			continue
		}

		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		eg.Go(func() error {
			defer sem.Release(1)

			if err := bc.Add(bdm.Height()); err != nil {
				if !xerrors.Is(err, util.NotFoundError) {
					return err
				}
			}

			return nil
		})
	}

	if err := sem.Acquire(ctx, limit); err != nil {
		if !xerrors.Is(err, context.Canceled) {
			return err
		}
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}
