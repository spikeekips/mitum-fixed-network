package mongodbstorage

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

type (
	getRecordCallback  func(*mongo.SingleResult) error
	getRecordsCallback func(*mongo.Cursor) (bool, error)
)

type Client struct {
	uri         string
	client      *mongo.Client
	db          *mongo.Database
	execTimeout time.Duration
}

func NewClient(uri string, connectTimeout, execTimeout time.Duration) (*Client, error) {
	var cs connstring.ConnString
	if c, err := checkURI(uri); err != nil {
		return nil, storage.WrapError(err)
	} else {
		cs = c
	}

	clientOpts := options.Client().ApplyURI(uri)
	if err := clientOpts.Validate(); err != nil {
		return nil, storage.WrapError(err)
	}

	var client *mongo.Client
	{
		ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
		defer cancel()

		if c, err := mongo.Connect(ctx, clientOpts); err != nil {
			return nil, storage.WrapError(xerrors.Errorf("connect timeout: %w", err))
		} else {
			client = c
		}
	}

	{
		ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
		defer cancel()

		if err := client.Ping(ctx, readpref.Primary()); err != nil {
			return nil, storage.WrapError(xerrors.Errorf("ping timeout: %w", err))
		}
	}

	return &Client{
		uri:         uri,
		client:      client,
		db:          client.Database(cs.Database),
		execTimeout: execTimeout,
	}, nil
}

func (cl *Client) Collection(col string) *mongo.Collection {
	return cl.db.Collection(col)
}

func (cl *Client) Find(
	col string,
	query interface{},
	callback getRecordsCallback,
	opts ...*options.FindOptions,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), cl.execTimeout)
	defer cancel()

	var cursor *mongo.Cursor
	if c, err := cl.db.Collection(col).Find(ctx, query, opts...); err != nil {
		return err
	} else {
		defer func() {
			_ = c.Close(context.TODO()) // TODO logging
		}()

		cursor = c
	}

	next := func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), cl.execTimeout)
		defer cancel()

		return cursor.Next(ctx)
	}

	var err error
	for next() {
		if keep, e := callback(cursor); e != nil {
			err = e
			break
		} else if !keep {
			break
		}
	}

	return err
}

func (cl *Client) GetByID(
	col string,
	id interface{},
	callback getRecordCallback,
	opts ...*options.FindOneOptions,
) error {
	res, err := cl.getByFilter(col, util.NewBSONFilter("_id", id).D(), opts...)
	if err != nil {
		return err
	}

	if callback == nil {
		return nil
	}

	return callback(res)
}

func (cl *Client) GetByFilter(
	col string,
	filter bson.D,
	callback getRecordCallback,
	opts ...*options.FindOneOptions,
) error {
	res, err := cl.getByFilter(col, filter, opts...)
	if err != nil {
		return err
	}

	if callback == nil {
		return nil
	}

	return callback(res)
}

func (cl *Client) getByFilter(col string, filter bson.D, opts ...*options.FindOneOptions) (*mongo.SingleResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cl.execTimeout)
	defer cancel()

	res := cl.db.Collection(col).FindOne(ctx, filter, opts...)
	if err := res.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, storage.NotFoundError.Wrap(err)
		}

		return nil, storage.WrapError(err)
	}

	return res, nil
}

func (cl *Client) Set(col string, doc Doc) (interface{}, error) {
	if doc.ID() == nil {
		return cl.setWithoutID(col, doc)
	}

	return cl.setWithID(col, doc)
}

func (cl *Client) setWithID(col string, doc Doc) (interface{}, error) {
	// NOTE remove existing one
	models := []mongo.WriteModel{
		mongo.NewDeleteOneModel().SetFilter(util.NewBSONFilter("_id", doc.ID()).D()),
		mongo.NewInsertOneModel().SetDocument(doc),
	}

	if err := cl.bulk(col, models); err != nil {
		return nil, err
	}

	return doc.ID(), nil
}

func (cl *Client) SetRaw(col string, raw bson.Raw) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cl.execTimeout)
	defer cancel()

	res, err := cl.db.Collection(col).InsertOne(ctx, raw)
	if err != nil {
		return nil, storage.WrapError(err)
	}

	return res.InsertedID, nil
}

func (cl *Client) setWithoutID(col string, doc interface{}) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cl.execTimeout)
	defer cancel()

	res, err := cl.db.Collection(col).InsertOne(ctx, doc)
	if err != nil {
		return nil, storage.WrapError(err)
	}

	return res.InsertedID, nil
}

func (cl *Client) bulk(col string, models []mongo.WriteModel) error {
	ctx, cancel := context.WithTimeout(context.Background(), cl.execTimeout)
	defer cancel()

	opts := options.BulkWrite().SetOrdered(true)
	res, err := cl.db.Collection(col).BulkWrite(ctx, models, opts)
	if err != nil {
		return storage.WrapError(err)
	} else if res.InsertedCount < 1 {
		return storage.WrapError(xerrors.Errorf("not inserted"))
	}

	return nil
}

func (cl *Client) Bulk(col string, models []mongo.WriteModel) error {
	return cl.bulk(col, models)
}

func (cl *Client) Count(col string, filter bson.D, opts ...*options.CountOptions) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cl.execTimeout)
	defer cancel()

	count, err := cl.db.Collection(col).CountDocuments(ctx, filter, opts...)

	return count, storage.WrapError(err)
}

func (cl *Client) Delete(col string, filter bson.D, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cl.execTimeout)
	defer cancel()

	return cl.db.Collection(col).DeleteMany(ctx, filter, opts...)
}

func (cl *Client) Exists(col string, filter bson.D) (bool, error) {
	count, err := cl.Count(col, filter, options.Count().SetLimit(1))

	return count > 0, err
}

func (cl *Client) WithSession(
	callback func(mongo.SessionContext, func(string /* collection */) *mongo.Collection) (interface{}, error),
) (interface{}, error) {
	opts := options.Session().SetDefaultReadConcern(readconcern.Majority())
	sess, err := cl.client.StartSession(opts)
	if err != nil {
		return nil, storage.WrapError(err)
	}
	defer sess.EndSession(context.TODO())

	txnOpts := options.Transaction().SetReadPreference(readpref.PrimaryPreferred())
	result, err := sess.WithTransaction(
		context.TODO(),
		func(sessCtx mongo.SessionContext) (interface{}, error) {
			return callback(sessCtx, cl.Collection)
		},
		txnOpts,
	)
	if err != nil {
		return nil, storage.WrapError(err)
	}

	return result, nil
}

func (cl *Client) DropDatabase() error {
	ctx, cancel := context.WithTimeout(context.Background(), cl.execTimeout)
	defer cancel()

	return cl.db.Drop(ctx)
}

func (cl *Client) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), cl.execTimeout)
	defer cancel()

	return cl.client.Disconnect(ctx)
}

func (cl *Client) Raw() *mongo.Client {
	return cl.client
}

func (cl *Client) CopyCollection(source *Client, fromCol, toCol string) error {
	var limit int = 100
	var models []mongo.WriteModel
	err := source.Find(fromCol, bson.D{}, func(cursor *mongo.Cursor) (bool, error) {
		if len(models) == limit {
			if err := cl.Bulk(toCol, models); err != nil {
				return false, err
			} else {
				models = nil
			}
		}

		raw := util.CopyBytes(cursor.Current)
		models = append(models, mongo.NewInsertOneModel().SetDocument(bson.Raw(raw)))

		return true, nil
	})
	if err != nil {
		return err
	}

	if len(models) < 1 {
		return nil
	}

	return cl.Bulk(toCol, models)
}
