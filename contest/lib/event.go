package contestlib

import (
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
)

var (
	reLookupKeyFormat = `^\w[a-zA-Z0-9_.]*$`
	reLookupKey       = regexp.MustCompile(reLookupKeyFormat)
)

type Event struct {
	m bson.M
}

func EmptyEvent() *Event {
	return &Event{
		m: bson.M{
			"_id":   util.ULID().String(),
			"level": "info",
			"t":     localtime.Now().Format(time.RFC3339Nano),
		},
	}
}

func NewEvent(b []byte) (*Event, error) {
	var m bson.M
	if b != nil {
		if err := jsonenc.Unmarshal(b, &m); err != nil {
			return nil, err
		}
	}
	m["_id"] = util.ULID().String()

	return &Event{m: m}, nil
}

func (li *Event) String() string {
	return string(li.Bytes())
}

func (li *Event) Bytes() []byte {
	b, err := bsonenc.Marshal(li.m)
	if err != nil {
		return nil
	}

	return b
}

func (li *Event) Add(key string, value interface{}) *Event {
	li.m[key] = value

	return li
}

func (li *Event) Map() map[string]interface{} {
	return li.m
}

func (li *Event) Raw() (bson.Raw, error) {
	if b, err := bsonenc.Marshal(li.m); err != nil {
		return nil, xerrors.Errorf("failed to unmarshal to bson.Raw in NewEventDoc: %w", err)
	} else {
		return b, nil
	}
}

func IsValidLookupKey(key string) bool {
	if !reLookupKey.Match([]byte(key)) {
		return false
	} else if strings.HasSuffix(key, ".") {
		return false
	}

	return true
}

func Lookup(o map[string]interface{}, key string) (interface{}, bool) {
	if !IsValidLookupKey(key) {
		return nil, false
	}

	ts := strings.SplitN(key, ".", -1)

	return lookupByKeys(o, ts)
}

func lookupByKeys(o map[string]interface{}, keys []string) (interface{}, bool) {
	var found bool
	var f interface{}
	for k, v := range o {
		if k != keys[0] {
			continue
		}

		f = v
		found = true
		break
	}

	if len(keys) == 1 {
		return f, found
	}

	if vv, ok := f.(map[string]interface{}); !ok {
		return nil, false
	} else {
		return lookupByKeys(vv, keys[1:])
	}
}
