package state

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

func (st *StateV0) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	key string,
	bValue []byte,
	previousBlock valuehash.Hash,
	height base.Height,
	currentBlock valuehash.Hash,
	bOperationInfos [][]byte,
) error {
	if h != nil && h.Empty() {
		return xerrors.Errorf("empty previous_block hash found")
	}

	if previousBlock.Empty() {
		previousBlock = nil
	}

	if currentBlock != nil && currentBlock.Empty() {
		return xerrors.Errorf("empty previous_block hash found")
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
	st.currentHeight = height
	st.currentBlock = currentBlock
	st.operations = ops

	return nil
}

func (oi *OperationInfoV0) unpack(_ encoder.Encoder, operation, seal valuehash.Hash) error {
	oi.oh = operation
	oi.sh = seal

	return nil
}
