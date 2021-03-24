package block

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (bd *BaseBlockDataMap) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	height base.Height,
	bh valuehash.Hash,
	createdAt time.Time,
	bitems map[string][]byte,
	writer hint.Hint,
) error {
	bd.h = h
	bd.height = height
	bd.block = bh
	bd.createdAt = createdAt
	bd.writerHint = writer

	items := map[string]BaseBlockDataMapItem{}
	for k := range bitems {
		if i, err := DecodeBaseBlockDataMapItem(enc, bitems[k]); err != nil {
			return err
		} else {
			items[k] = i
		}
	}

	bd.items = items

	return nil
}

func (bd *BaseBlockDataMapItem) unpack(dataType, checksum, url string) error {
	bd.t = dataType
	bd.checksum = checksum
	bd.url = url

	return nil
}

func DecodeBaseBlockDataMapItem(enc encoder.Encoder, b []byte) (BaseBlockDataMapItem, error) {
	var ubdi BaseBlockDataMapItem
	if err := enc.Decode(b, &ubdi); err != nil {
		return ubdi, err
	}

	return ubdi, nil
}
