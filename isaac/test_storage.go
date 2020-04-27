// +build test

package isaac

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/storage"
	leveldbstorage "github.com/spikeekips/mitum/storage/leveldb"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util/encoder"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

type StorageSupportTest struct {
	DBType  string
	Encs    *encoder.Encoders
	JSONEnc *jsonencoder.Encoder
	BSONEnc *bsonencoder.Encoder
}

func (ss *StorageSupportTest) SetupSuite() {
	ss.Encs = encoder.NewEncoders()

	ss.JSONEnc = jsonencoder.NewEncoder()
	_ = ss.Encs.AddEncoder(ss.JSONEnc)

	ss.BSONEnc = bsonencoder.NewEncoder()
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

		if enc == nil {
			enc = ss.BSONEnc
		}

		return DummyMongodbStorage{mongodbstorage.NewStorage(client, encs, enc)}
	default:
		panic(xerrors.Errorf("unknown db type: %v", ss.DBType))
	}
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
