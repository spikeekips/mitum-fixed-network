package deploy

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
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
			network.WriteProblemWithError(w, http.StatusBadRequest,
				errors.Wrap(err, "failed to load blockdatamaps"))
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

		b, err := jsonenc.Marshal(map[string]interface{}{
			"heights":       heights,
			"expected_time": localtime.UTCNow().Add(bc.RemoveAfter()),
		})
		if err != nil {
			network.WriteProblemWithError(w, http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		_, _ = w.Write(b)
	}
}

func loadBlockDataMaps(r *http.Request, enc encoder.Encoder) ([]block.BlockDataMap, error) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var hinters []hint.Hinter
	switch j, err := enc.DecodeSlice(b); {
	case err != nil:
		return nil, err
	case len(j) < 1:
		return nil, errors.Errorf("empty blockdatamaps")
	default:
		hinters = j
	}

	founds := map[base.Height]bool{}
	ubd := make([]block.BlockDataMap, len(hinters))
	for i := range hinters {
		j := hinters[i]
		if k, ok := j.(block.BlockDataMap); !ok {
			return nil, util.WrongTypeError.Errorf("not block.BlockDataMap type, %T", j)
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
	wk := util.NewErrgroupWorker(context.Background(), 100)
	defer wk.Close()

	go func() {
		defer wk.Done()

		for i := range bdms {
			bdm := bdms[i]
			if err := wk.NewJob(func(context.Context, uint64) error {
				return checkBlockDataMap(db, bdm)
			}); err != nil {
				return
			}
		}
	}()

	return wk.Wait()
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
		return errors.Errorf("block hash does not match with manifest")
	}

	return nil
}

func commitBlockDataMaps(db storage.Database, bc *BlockDataCleaner, bdms []block.BlockDataMap) error {
	if err := db.SetBlockDataMaps(bdms); err != nil {
		return err
	}

	wk := util.NewErrgroupWorker(context.Background(), 100)
	defer wk.Close()

	go func() {
		defer wk.Done()

		for i := range bdms {
			bdm := bdms[i]
			if bdm.IsLocal() { // NOTE local blockdata will not be removed
				continue
			}

			if err := wk.NewJob(func(context.Context, uint64) error {
				if err := bc.Add(bdm.Height()); err != nil {
					if !errors.Is(err, util.NotFoundError) {
						return err
					}
				}

				return nil
			}); err != nil {
				return
			}
		}
	}()

	return wk.Wait()
}
