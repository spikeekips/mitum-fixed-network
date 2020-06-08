// +build test

package isaac

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	leveldbstorage "github.com/spikeekips/mitum/storage/leveldb"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type StorageSupportTest struct {
	DBType  string
	Encs    *encoder.Encoders
	JSONEnc *jsonenc.Encoder
	BSONEnc *bsonenc.Encoder
}

func (ss *StorageSupportTest) SetupSuite() {
	ss.Encs = encoder.NewEncoders()

	ss.JSONEnc = jsonenc.NewEncoder()
	_ = ss.Encs.AddEncoder(ss.JSONEnc)

	ss.BSONEnc = bsonenc.NewEncoder()
	_ = ss.Encs.AddEncoder(ss.BSONEnc)
}

func (ss *StorageSupportTest) Storage(encs *encoder.Encoders, enc encoder.Encoder) storage.Storage {
	if encs == nil {
		encs = ss.Encs
	}

	if len(ss.DBType) < 1 {
		ss.DBType = "leveldb"
	}

	switch ss.DBType {
	case "leveldb":
		if enc == nil {
			enc = ss.JSONEnc
		}

		return leveldbstorage.NewMemStorage(encs, enc)
	case "mongodb":
		client, err := mongodbstorage.NewClient(mongodbstorage.TestMongodbURI(), time.Second*2, time.Second*2)
		if err != nil {
			panic(err)
		}

		if enc == nil || enc.Hint().Type() != bsonenc.BSONType {
			enc = ss.BSONEnc
		}

		st, err := mongodbstorage.NewStorage(client, encs, enc)
		if err != nil {
			panic(err)
		}

		d := DummyMongodbStorage{st}

		_ = d.Initialize()

		return d
	default:
		panic(xerrors.Errorf("unknown db type: %v", ss.DBType))
	}
}

func (ss *StorageSupportTest) SetBlock(st storage.Storage, blk block.Block) error {
	if bs, err := st.OpenBlockStorage(blk); err != nil {
		return err
	} else if err := bs.Commit(); err != nil {
		return err
	}

	return nil
}

type DummyMongodbStorage struct {
	*mongodbstorage.Storage
}

func (dm DummyMongodbStorage) Close() error {
	if err := dm.Client().DropDatabase(); err != nil {
		return err
	}

	return dm.Storage.Close()
}
