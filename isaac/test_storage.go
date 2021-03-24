// +build test

package isaac

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/storage"
	leveldbstorage "github.com/spikeekips/mitum/storage/leveldb"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
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
		c := mongoConnPool.Get().(*mongodbstorage.Client)
		client, err := c.New(fmt.Sprintf("t-%s", util.UUID().String()))
		if err != nil {
			panic(err)
		}

		if enc == nil || enc.Hint().Type() != bsonenc.BSONType {
			enc = ss.BSONEnc
		}

		st, err := mongodbstorage.NewStorage(client, encs, enc, cache.Dummy{})
		if err != nil {
			panic(err)
		}

		d := NewDummyMongodbStorage(st)

		_ = d.Initialize()

		return d
	case "mongodb+gcache":
		c := mongoConnPool.Get().(*mongodbstorage.Client)
		client, err := c.New(fmt.Sprintf("t-%s", util.UUID().String()))
		if err != nil {
			panic(err)
		}

		if enc == nil || enc.Hint().Type() != bsonenc.BSONType {
			enc = ss.BSONEnc
		}

		ca, err := cache.NewGCache("lru", 100*100, time.Hour)
		if err != nil {
			panic(err)
		}
		st, err := mongodbstorage.NewStorage(client, encs, enc, ca)
		if err != nil {
			panic(err)
		}

		d := NewDummyMongodbStorage(st)

		_ = d.Initialize()

		return d
	default:
		panic(xerrors.Errorf("unknown db type: %v", ss.DBType))
	}
}

type DummyMongodbStorage struct {
	*mongodbstorage.Storage
}

func NewDummyMongodbStorage(st *mongodbstorage.Storage) DummyMongodbStorage {
	d := DummyMongodbStorage{Storage: st}
	if err := d.Initialize(); err != nil {
		panic(err)
	}

	return d
}

func (dm DummyMongodbStorage) Close() error {
	if err := dm.Client().DropDatabase(); err != nil {
		return err
	}

	mongoConnPool.Put(dm.Client())

	return dm.Storage.Close()
}

var mongoConnPool = sync.Pool{
	New: func() interface{} {
		client, err := mongodbstorage.NewClient(mongodbstorage.TestMongodbURI(), time.Second*2, time.Second*2)
		if err != nil {
			panic(err)
		}

		return client
	},
}
