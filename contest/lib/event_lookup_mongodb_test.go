// +build mongodb

package contestlib

import (
	"bytes"
	"html/template"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func decodeQuery(b []byte) (bson.M, error) {
	var q bson.M
	err := bson.UnmarshalExtJSON(b, false, &q)

	return q, err
}

func TestLookupMongodb(t *testing.T) {
	cases := []struct {
		name  string
		doc   bson.M
		query string
		args  map[string]interface{}
	}{
		{
			name:  "simple",
			doc:   bson.M{"a": 1, "b": 2},
			query: `{"a": 1}`,
		},
		{
			name:  "by _id",
			doc:   bson.M{"a": 1, "b": 2, "_id": "0000XSNJG0MQJHBF4QX1EFD6Y3"},
			query: `{"_id": "{{._id}}"}`,
			args:  map[string]interface{}{"_id": "0000XSNJG0MQJHBF4QX1EFD6Y3"},
		},
		{
			name:  "by int",
			doc:   bson.M{"a": 1, "b": 2},
			query: `{"b": {{.b}}}`,
			args:  map[string]interface{}{"b": 2},
		},
		{
			name:  "by bool",
			doc:   bson.M{"a": 1, "b": 2, "c": true},
			query: `{"c": {{.c}}}`,
			args:  map[string]interface{}{"c": true},
		},
		{
			name:  "by bool: false",
			doc:   bson.M{"a": 1, "b": 2, "c": false},
			query: `{"c": {{.c}}}`,
			args:  map[string]interface{}{"c": false},
		},
	}

	client, err := mongodbstorage.NewClient(mongodbstorage.TestMongodbURI(), time.Second*2, time.Second*2)
	assert.NoError(t, err)

	defer func() {
		_ = client.DropDatabase()
	}()

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func(*testing.T) {
				r, err := bsonenc.Marshal(c.doc)
				assert.NoError(t, err, "%d: %v", i, c.name)

				id, err := client.AddRaw("test", r)
				assert.NoError(t, err, "%d: %v", i, c.name)

				args := map[string]interface{}{
					"_id": id,
				}

				for k, v := range c.args {
					args[k] = v
				}

				var bf bytes.Buffer
				qt, err := template.New("query").Parse(c.query)
				assert.NoError(t, err, "%d: %v", i, c.name)
				qt.Execute(&bf, args)

				q, err := decodeQuery(bf.Bytes())
				assert.NoError(t, err, "%d: %v", i, c.name)

				var doc bson.M
				assert.NoError(t, client.Find(nil, "test", q, func(cursor *mongo.Cursor) (bool, error) {
					assert.NoError(t, cursor.Decode(&doc))

					return false, nil
				},
					options.Find().SetSort(bson.M{"_id": -1}),
				), "%d: %v", i, c.name)
				assert.NotNil(t, doc, "%d: %v", i, c.name)

				var _id interface{}
				var doc_id interface{}
				switch f := id.(type) {
				case primitive.ObjectID:
					_id = f.Hex()
					did, ok := doc["_id"].(primitive.ObjectID)
					doc_id = did.Hex()
					assert.True(t, ok, "%d: %v", i, c.name)
				case string:
					_id = f
					did, ok := doc["_id"].(string)
					doc_id = did
					assert.True(t, ok, "%d: %v", i, c.name)
				}

				assert.Equal(t, _id, doc_id, "%d: %v", i, c.name)
			},
		)
	}
}
