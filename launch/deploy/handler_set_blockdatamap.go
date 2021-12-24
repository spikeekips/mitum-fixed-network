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

var LimitBlockdataMaps = 100

func NewSetBlockdataMapsHandler(
	enc encoder.Encoder,
	db storage.Database,
	bc *BlockdataCleaner,
) network.HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var bdms []block.BlockdataMap
		switch i, err := loadBlockdataMaps(r, enc); {
		case err != nil:
			network.WriteProblemWithError(w, http.StatusBadRequest,
				errors.Wrap(err, "failed to load blockdatamaps"))
			return
		case len(i) < 1, len(i) > LimitBlockdataMaps:
			network.WriteProblemWithError(w, http.StatusBadRequest, err)
			return
		default:
			bdms = i
		}

		if err := checkBlockdataMaps(db, bdms); err != nil {
			network.WriteProblemWithError(w, http.StatusBadRequest, err)
			return
		}

		if err := commitBlockdataMaps(db, bc, bdms); err != nil {
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

func loadBlockdataMaps(r *http.Request, enc encoder.Encoder) ([]block.BlockdataMap, error) {
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
	ubd := make([]block.BlockdataMap, len(hinters))
	for i := range hinters {
		j := hinters[i]
		if k, ok := j.(block.BlockdataMap); !ok {
			return nil, util.WrongTypeError.Errorf("not block.BlockdataMap type, %T", j)
		} else if _, found := founds[k.Height()]; found {
			continue
		} else {
			ubd[i] = k
			founds[k.Height()] = true
		}
	}

	return ubd, nil
}

func checkBlockdataMaps(db storage.Database, bdms []block.BlockdataMap) error {
	wk := util.NewErrgroupWorker(context.Background(), 100)
	defer wk.Close()

	go func() {
		defer wk.Done()

		for i := range bdms {
			bdm := bdms[i]
			if err := wk.NewJob(func(context.Context, uint64) error {
				return checkBlockdataMap(db, bdm)
			}); err != nil {
				return
			}
		}
	}()

	return wk.Wait()
}

func checkBlockdataMap(db storage.Database, bdm block.BlockdataMap) error {
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

func commitBlockdataMaps(db storage.Database, bc *BlockdataCleaner, bdms []block.BlockdataMap) error {
	if err := db.SetBlockdataMaps(bdms); err != nil {
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
