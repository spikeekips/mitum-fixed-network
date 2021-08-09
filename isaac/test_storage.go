// +build test

package isaac

import (
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/storage"
	leveldbstorage "github.com/spikeekips/mitum/storage/leveldb"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
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
	dbs     []storage.Database
}

func (ss *StorageSupportTest) SetupSuite() {
	ss.Encs = encoder.NewEncoders()

	ss.JSONEnc = jsonenc.NewEncoder()
	_ = ss.Encs.AddEncoder(ss.JSONEnc)

	ss.BSONEnc = bsonenc.NewEncoder()
	_ = ss.Encs.AddEncoder(ss.BSONEnc)
}

func (ss *StorageSupportTest) Database(encs *encoder.Encoders, enc encoder.Encoder) storage.Database {
	d := ss.database(encs, enc)
	ss.dbs = append(ss.dbs, d)

	return d
}

func (ss *StorageSupportTest) TearDownTest() {
	for i := range ss.dbs {
		_ = ss.dbs[i].Close()
	}
}

func (ss *StorageSupportTest) database(encs *encoder.Encoders, enc encoder.Encoder) storage.Database {
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

		return leveldbstorage.NewMemDatabase(encs, enc)
	case "mongodb":
		client, err := mongodbstorage.NewClient(mongodbstorage.TestMongodbURI(), time.Second*2, time.Second*2)
		if err != nil {
			panic(err)
		}

		if enc == nil || enc.Hint().Type() != bsonenc.BSONEncoderType {
			enc = ss.BSONEnc
		}

		st, err := mongodbstorage.NewDatabase(client, encs, enc, cache.Dummy{})
		if err != nil {
			panic(err)
		}

		d := NewDummyMongodbDatabase(st)

		_ = d.Initialize()

		return d
	case "mongodb+gcache":
		client, err := mongodbstorage.NewClient(mongodbstorage.TestMongodbURI(), time.Second*2, time.Second*2)
		if err != nil {
			panic(err)
		}

		if enc == nil || enc.Hint().Type() != bsonenc.BSONEncoderType {
			enc = ss.BSONEnc
		}

		ca, err := cache.NewGCache("lru", 100*100, time.Hour)
		if err != nil {
			panic(err)
		}
		st, err := mongodbstorage.NewDatabase(client, encs, enc, ca)
		if err != nil {
			panic(err)
		}

		d := NewDummyMongodbDatabase(st)

		_ = d.Initialize()

		return d
	default:
		panic(errors.Errorf("unknown db type: %v", ss.DBType))
	}
}

type DummyMongodbDatabase struct {
	*mongodbstorage.Database
}

func NewDummyMongodbDatabase(st *mongodbstorage.Database) DummyMongodbDatabase {
	d := DummyMongodbDatabase{Database: st}
	if err := d.Initialize(); err != nil {
		panic(err)
	}

	return d
}

func (dm DummyMongodbDatabase) Close() error {
	if err := dm.Client().DropDatabase(); err != nil {
		return err
	}

	return dm.Database.Close()
}
