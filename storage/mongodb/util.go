package mongodbstorage

import (
	"fmt"
	"net/url"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

type Doc interface {
	ID() interface{}
}

func encodeWithEncoder(enc encoder.Encoder, i interface{}) (bson.M, error) {
	var data []byte
	if b, err := enc.Encode(i); err != nil {
		return nil, err
	} else {
		data = b
	}

	var h []byte
	if b, err := util.JSONMarshal(enc.Hint()); err != nil {
		return nil, err
	} else {
		h = b
	}

	return bson.M{
		"encoder": h,
		"data":    data,
	}, nil
}

func checkURI(uri string) (connstring.ConnString, error) {
	cs, err := connstring.Parse(uri)
	if err != nil {
		return connstring.ConnString{}, storage.WrapError(err)
	}

	if len(cs.Database) < 1 {
		return connstring.ConnString{}, storage.WrapError(xerrors.Errorf("empty database name in mongodb uri"))
	}

	return cs, nil
}

func updateDBInURI(uri, db string) (string, error) {
	if _, err := checkURI(uri); err != nil {
		return "", err
	}

	var n string
	if parsed, err := url.Parse(uri); err != nil {
		return "", err
	} else {
		parsed.Path = db
		n = parsed.String()
	}

	return n, nil
}

func NewTempURI(uri, prefix string) (string, error) {
	tmp := fmt.Sprintf("%s_%s", prefix, util.UUID().String())

	cs, err := updateDBInURI(uri, tmp)
	if err != nil {
		return "", err
	}

	return cs, nil
}

func BSONE(key string, value interface{}) bson.E {
	return bson.E{Key: key, Value: value}
}

type Filter struct {
	d bson.D
}

func EmptyFilter() *Filter {
	return &Filter{d: bson.D{}}
}

func NewFilter(key string, value interface{}) *Filter {
	ft := EmptyFilter()

	return ft.Add(key, value)
}

func (ft *Filter) Add(key string, value interface{}) *Filter {
	ft.d = append(ft.d, bson.E{Key: key, Value: value})

	return ft
}

func (ft *Filter) D() bson.D {
	return ft.d
}
