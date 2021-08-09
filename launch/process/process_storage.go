package process

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

const (
	ProcessNameDatabase  = "database"
	ProcessNameBlockData = "blockdata"
)

var (
	ProcessorBlockData pm.Process
	ProcessorDatabase  pm.Process
)

func init() {
	if i, err := pm.NewProcess(
		ProcessNameDatabase,
		[]string{
			ProcessNameConfig,
		},
		ProcessMongodbDatabase,
	); err != nil {
		panic(err)
	} else {
		ProcessorDatabase = i
	}

	if i, err := pm.NewProcess(
		ProcessNameBlockData,
		[]string{
			ProcessNameDatabase,
		},
		ProcessBlockData,
	); err != nil {
		panic(err)
	} else {
		ProcessorBlockData = i
	}
}

func ProcessBlockData(ctx context.Context) (context.Context, error) {
	var l config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	}
	conf := l.Storage().BlockData()

	var enc *jsonenc.Encoder
	if err := config.LoadJSONEncoderContextValue(ctx, &enc); err != nil {
		return ctx, err
	}

	blockData := localfs.NewBlockData(conf.Path(), enc)
	if err := blockData.Initialize(); err != nil {
		return nil, err
	}

	var forceCreate bool
	if err := LoadGenesisBlockForceCreateContextValue(ctx, &forceCreate); err != nil {
		if !errors.Is(err, util.ContextValueNotFoundError) {
			return ctx, err
		}
	}

	return context.WithValue(ctx, ContextValueBlockData, blockData), nil
}

func ProcessMongodbDatabase(ctx context.Context) (context.Context, error) {
	var l config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	}
	conf := l.Storage().Database()

	if !strings.EqualFold(conf.URI().Scheme, "mongodb") {
		return ctx, nil
	}

	ca, err := cache.NewCacheFromURI(conf.Cache().String())
	if err != nil {
		return ctx, err
	}

	var encs *encoder.Encoders
	if err := config.LoadEncodersContextValue(ctx, &encs); err != nil {
		return ctx, err
	}

	if st, err := mongodbstorage.NewDatabaseFromURI(conf.URI().String(), encs, ca); err != nil {
		return ctx, err
	} else if err := st.Initialize(); err != nil {
		return ctx, err
	} else {
		return context.WithValue(ctx, ContextValueDatabase, st), nil
	}
}
