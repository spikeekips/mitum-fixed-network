// +build mongodb

package contestlib

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
)

type testEventMongodb struct {
	suite.Suite
	client *mongodbstorage.Client
}

func (t *testEventMongodb) SetupTest() {
	client, err := mongodbstorage.NewClient(mongodbstorage.TestMongodbURI(), time.Second*2, time.Second*2)
	if err != nil {
		panic(err)
	}

	t.client = client
}

func (t *testEventMongodb) TearDownTest() {
	t.client.Collection("test").Drop(context.TODO())
}

func (t *testEventMongodb) TestInsert() {
	b := `{
  "level": "info",
  "module": "consensus-states",
  "handler": 33,
  "t": "2020-05-21T08:19:45.76229515Z",
  "m": "activated: JOINING"
}`
	e, err := NewEvent([]byte(b))
	t.NoError(err)

	r, err := e.Raw()
	t.NoError(err)

	id, err := t.client.SetRaw("test", r)
	t.NoError(err)

	var event map[string]interface{}
	t.NoError(t.client.GetByID("test", id, func(res *mongo.SingleResult) error {
		return res.Decode(&event)
	}))

	t.Equal("info", event["level"].(string))
	t.Equal("consensus-states", event["module"].(string))
	t.Equal(int32(33), event["handler"].(int32))
	t.Equal("2020-05-21T08:19:45.76229515Z", event["t"].(string))
	t.Equal("activated: JOINING", event["m"].(string))
}

func (t *testEventMongodb) TestPreseveInsertedOrder() {
	temp := `{
  "seq": %d
}`

	for i := 0; i < 10; i++ {
		e, err := NewEvent([]byte(fmt.Sprintf(temp, i)))
		t.NoError(err)

		r, err := e.Raw()
		t.NoError(err)

		_, err = t.client.SetRaw("test", r)
		t.NoError(err)
	}

	count, err := t.client.Count("test", bson.D{})
	t.NoError(err)
	t.Equal(int64(10), count)

	var seqs []int64
	t.client.Find("test", bson.D{}, func(cursor *mongo.Cursor) (bool, error) {
		s := cursor.Current.Lookup("seq").Int32()
		seqs = append(seqs, int64(s))

		return true, nil
	})

	for i := int64(0); i < 10; i++ {
		t.Equal(i, seqs[i])
	}
}

func TestEventMongodb(t *testing.T) {
	suite.Run(t, new(testEventMongodb))
}
