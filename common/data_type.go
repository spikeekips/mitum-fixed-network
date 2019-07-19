package common

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
)

type DataType struct {
	id   uint
	name string
}

func NewDataType(id uint, name string) DataType {
	if id < 1 {
		panic(fmt.Errorf("DataType.id should be greater than 0"))
	}

	return DataType{id: id, name: name}
}

func (i DataType) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.name)
}

func (i DataType) ID() uint {
	return i.id
}

func (i DataType) Name() string {
	return i.name
}

func (i DataType) Equal(b DataType) bool {
	return i.id == b.id
}

func (i DataType) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, struct {
		ID   uint
		Name string
	}{
		ID:   i.id,
		Name: i.name,
	})
}

func (i *DataType) DecodeRLP(s *rlp.Stream) error {
	var d struct {
		ID   uint
		Name string
	}
	if err := s.Decode(&d); err != nil {
		return err
	}

	i.id = d.ID
	i.name = d.Name

	return nil
}

func (i DataType) Empty() bool {
	return i.id < 1
}

func (i DataType) String() string {
	return i.name
}
