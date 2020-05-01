// +build mongodb

package mongodbstorage

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
)

type docNilID struct {
	id interface{}
	bson.M
}

func NewDocNilID(id interface{}, m bson.M) docNilID {
	return docNilID{id: id, M: m}
}

func (doc docNilID) ID() interface{} {
	return doc.id
}

func (doc docNilID) MarshalBSON() ([]byte, error) {
	if doc.id != nil {
		doc.M["_id"] = doc.id
	}

	return bsonencoder.Marshal(doc.M)
}

func (doc docNilID) Doc() bson.M {
	return doc.M
}

type testMongodbClient struct {
	suite.Suite
	encs   *encoder.Encoders
	enc    encoder.Encoder
	client *Client
}

func (t *testMongodbClient) SetupSuite() {
	t.encs = encoder.NewEncoders()
	t.enc = bsonencoder.NewEncoder()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.AddHinter(key.BTCPublickey{})
	_ = t.encs.AddHinter(block.BlockV0{})
	_ = t.encs.AddHinter(block.ManifestV0{})
	_ = t.encs.AddHinter(block.BlockConsensusInfoV0{})
	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(base.VoteproofV0{})
	_ = t.encs.AddHinter(seal.DummySeal{})
	_ = t.encs.AddHinter(operation.Seal{})
	_ = t.encs.AddHinter(operation.KVOperation{})
	_ = t.encs.AddHinter(operation.KVOperationFact{})
}

func (t *testMongodbClient) SetupTest() {
	client, err := NewClient(TestMongodbURI(), time.Second*2, time.Second*2)
	t.NoError(err)

	t.client = client
}

func (t *testMongodbClient) TearDownTest() {
	t.client.DropDatabase()
}

func (t *testMongodbClient) TestClient() {
	_, err := NewClient(TestMongodbURI(), time.Second*2, 0)
	t.NoError(err)
}

func (t *testMongodbClient) TestWrongURI() {
	_, err := NewClient("mongodb://222.222.222.222/ttt", time.Millisecond*10, 0)
	t.Contains(err.Error(), "context deadline exceeded")
}

func (t *testMongodbClient) TestWithoutDBName() {
	_, err := NewClient("mongodb://222.222.222.222", time.Millisecond*10, 0)
	t.Contains(err.Error(), "empty database name")
}

func (t *testMongodbClient) TestFindUnknown() {
	var records []bson.M

	err := t.client.Find(
		"showme",
		util.NewBSONFilter("findme", 1).D(),
		func(cursor *mongo.Cursor) (bool, error) {
			var record bson.M
			if err := cursor.Decode(&record); err != nil {
				return false, err
			} else {
				records = append(records, record)
			}

			return true, nil
		},
	)
	t.NoError(err)

	t.Equal(0, len(records))
}

func (t *testMongodbClient) TestInsertOne() {
	doc := NewDocNilID(nil, bson.M{"findme": int64(3)})

	inserted, err := t.client.Set("showme", doc)
	t.NoError(err)
	t.IsType(primitive.ObjectID{}, inserted)
	t.False(inserted.(primitive.ObjectID).IsZero())

	var rs []bson.M
	err = t.client.Find("showme", util.NewBSONFilter("findme", int64(3)).D(),
		func(cursor *mongo.Cursor) (bool, error) {
			var record bson.M
			if err := cursor.Decode(&record); err != nil {
				return false, err
			} else {
				rs = append(rs, record)
			}

			return true, nil
		},
	)
	t.NoError(err)

	t.Equal(1, len(rs))

	t.Equal(doc.Doc()["findme"], rs[0]["findme"])
}

func (t *testMongodbClient) TestOverwrite() {
	doc := NewDocNilID(nil, bson.M{"findme": int64(3)})

	id, err := t.client.Set("showme", doc)
	t.NoError(err)

	newDoc := NewDocNilID(id, bson.M{"findme": int64(33)})
	{
		inserted, err := t.client.Set("showme", newDoc)
		t.NoError(err)
		t.NotNil(inserted)

		t.Equal(id, inserted)
	}

	{ // existing one should be removed
		var rs []bson.M
		err := t.client.Find("showme", util.NewBSONFilter("findme", int64(3)).D(),
			func(cursor *mongo.Cursor) (bool, error) {
				var record bson.M
				if err := cursor.Decode(&record); err != nil {
					return false, err
				} else {
					rs = append(rs, record)
				}

				return true, nil
			},
		)
		t.NoError(err)

		t.Equal(0, len(rs))
	}

	var rs []bson.M
	err = t.client.Find("showme", util.NewBSONFilter("findme", int64(33)).D(),
		func(cursor *mongo.Cursor) (bool, error) {
			var record bson.M
			if err := cursor.Decode(&record); err != nil {
				return false, err
			} else {
				rs = append(rs, record)
			}

			return true, nil
		},
	)
	t.NoError(err)

	t.Equal(1, len(rs))

	t.Equal(newDoc.Doc()["findme"], rs[0]["findme"])
}

func (t *testMongodbClient) TestInsertWithObjectID() {
	// with long enough string based id
	id := fmt.Sprintf("%s-%s", valuehash.RandomSHA256().String(), valuehash.RandomSHA256().String())

	doc := NewDocNilID(id, bson.M{"findme": int64(3), "_id": id})
	inserted, err := t.client.Set("showme", doc)
	t.NoError(err)
	t.IsType("", inserted)
	t.Equal(id, inserted)

	var rs []bson.M
	err = t.client.Find("showme", util.NewBSONFilter("_id", id).D(),
		func(cursor *mongo.Cursor) (bool, error) {
			var record bson.M
			if err := cursor.Decode(&record); err != nil {
				return false, err
			} else {
				rs = append(rs, record)
			}

			return true, nil
		},
	)
	t.NoError(err)

	t.Equal(id, rs[0]["_id"])
	t.Equal(doc.Doc()["findme"], rs[0]["findme"])
}

func (t *testMongodbClient) TestMoveRawBytes() {
	doc := NewDocNilID(nil, bson.M{"findme": int64(3)})
	insertedID, err := t.client.Set("showme", doc)
	t.NoError(err)

	var newIsertedID interface{}
	t.client.Find("showme", bson.D{}, func(cursor *mongo.Cursor) (bool, error) {
		i, err := t.client.SetRaw("new_showme", cursor.Current)
		t.NoError(err)

		newIsertedID = i

		return false, nil
	})

	var newDoc bson.M
	err = t.client.GetByID("new_showme", newIsertedID,
		func(res *mongo.SingleResult) error {
			return res.Decode(&newDoc)
		})
	t.NoError(err)

	t.Equal(insertedID, newDoc["_id"])
	t.Equal(doc.Doc()["findme"], newDoc["findme"])
}

func TestMongodbClient(t *testing.T) {
	suite.Run(t, new(testMongodbClient))
}
