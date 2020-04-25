package state

import (
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
)

func (st *StateV0) unpack(
	enc encoder.Encoder,
	bH []byte,
	key string,
	bValue,
	bPreviousBlock []byte,
	bOperationInfos [][]byte,
) error {
	var h, previousBlock valuehash.Hash
	if i, err := valuehash.Decode(enc, bH); err != nil {
		return err
	} else {
		h = i
	}

	if i, err := valuehash.Decode(enc, bPreviousBlock); err != nil {
		return err
	} else {
		previousBlock = i
	}

	var value Value
	if v, err := DecodeValue(enc, bValue); err != nil {
		return err
	} else {
		value = v
	}

	ops := make([]OperationInfo, len(bOperationInfos))
	for i := range bOperationInfos {
		if oi, err := DecodeOperationInfo(enc, bOperationInfos[i]); err != nil {
			return err
		} else {
			ops[i] = oi
		}
	}

	st.h = h
	st.key = key
	st.value = value
	st.previousBlock = previousBlock
	st.operations = ops

	return nil
}

func (oi *OperationInfoV0) unpack(enc encoder.Encoder, bOperation, bSeal []byte) error {
	var oh, sh valuehash.Hash
	if h, err := valuehash.Decode(enc, bOperation); err != nil {
		return err
	} else {
		oh = h
	}

	if h, err := valuehash.Decode(enc, bSeal); err != nil {
		return err
	} else {
		sh = h
	}

	oi.oh = oh
	oi.sh = sh

	return nil
}
