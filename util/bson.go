package util

import (
	"go.mongodb.org/mongo-driver/bson"
)

type BSONFilter struct {
	d bson.D
}

func EmptyBSONFilter() *BSONFilter {
	return &BSONFilter{d: bson.D{}}
}

func NewBSONFilter(key string, value interface{}) *BSONFilter {
	ft := EmptyBSONFilter()

	return ft.Add(key, value)
}

func NewBSONFilterFromD(d bson.D) *BSONFilter {
	return &BSONFilter{d: d}
}

func (ft *BSONFilter) Add(key string, value interface{}) *BSONFilter {
	ft.d = append(ft.d, bson.E{Key: key, Value: value})

	return ft
}

func (ft *BSONFilter) AddOp(key string, value interface{}, op string) *BSONFilter {
	return ft.Add(key, bson.D{bson.E{Key: op, Value: value}})
}

func (ft *BSONFilter) D() bson.D {
	return ft.d
}
