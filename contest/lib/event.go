package contestlib

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
)

type Event struct {
	b []byte
	m map[string]interface{}
}

func EmptyEvent() *Event {
	return &Event{
		m: map[string]interface{}{
			"level": "info",
			"t":     localtime.Now().Format(time.RFC3339Nano),
		},
	}
}

func NewEvent(b []byte) (*Event, error) {
	m := map[string]interface{}{}
	if b != nil {
		if err := jsonencoder.Unmarshal(b, &m); err != nil {
			return nil, err
		}
	}

	return &Event{b: b, m: m}, nil
}

func NewEventFromMap(m map[string]interface{}) (*Event, error) {
	return &Event{m: m}, nil
}

func (li *Event) String() string {
	return string(li.b)
}

func (li *Event) Bytes() []byte {
	return li.b
}

func (li *Event) Add(key string, value interface{}) *Event {
	li.m[key] = value

	b, err := jsonencoder.Marshal(li.m)
	if err != nil {
		return nil
	}
	li.b = b

	return li
}

func (li *Event) Map() map[string]interface{} {
	return li.m
}

func (li *Event) Raw() (bson.Raw, error) {
	var r bson.Raw
	if err := bson.UnmarshalExtJSON(li.b, true, &r); err != nil {
		return nil, xerrors.Errorf("failed to unmarshal to bson.Raw in NewEventDoc: %w", err)
	}

	return r, nil
}
