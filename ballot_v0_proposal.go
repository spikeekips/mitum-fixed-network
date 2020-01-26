package mitum

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

var ProposalV0Hint hint.Hint = hint.MustHint(ProposalBallotType, "0.1")

type ProposalV0 struct {
	BaseBallotV0
	h  valuehash.Hash
	bh valuehash.Hash

	//-x-------------------- hashing parts
	seals []valuehash.Hash
	//--------------------x-
}

func (pb ProposalV0) Hint() hint.Hint {
	return ProposalV0Hint
}

func (pb ProposalV0) Stage() Stage {
	return StageProposal
}

func (pb ProposalV0) Hash() valuehash.Hash {
	return pb.h
}

func (pb ProposalV0) BodyHash() valuehash.Hash {
	return pb.bh
}

func (pb ProposalV0) IsValid(b []byte) error {
	if err := pb.BaseBallotV0.IsValid(b); err != nil {
		return err
	}

	for _, h := range pb.seals {
		if err := h.IsValid(b); err != nil {
			return err
		}
	}

	return nil
}

func (pb ProposalV0) Seals() []valuehash.Hash {
	return pb.seals
}

func (pb ProposalV0) GenerateHash(b []byte) (valuehash.Hash, error) {
	if err := pb.IsValid(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		pb.BaseBallotV0.Bytes(),
		func() []byte {
			var hl [][]byte
			for _, h := range pb.seals {
				hl = append(hl, h.Bytes())
			}

			return util.ConcatSlice(hl)
		}(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (pb ProposalV0) GenerateBodyHash(b []byte) (valuehash.Hash, error) {
	if err := pb.BaseBallotV0.IsReadyToSign(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		pb.BaseBallotV0.BodyBytes(),
		func() []byte {
			var hl [][]byte
			for _, h := range pb.seals {
				hl = append(hl, h.Bytes())
			}

			return util.ConcatSlice(hl)
		}(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (pb *ProposalV0) Sign(pk key.Privatekey, b []byte) error {
	var bodyHash valuehash.Hash
	if h, err := pb.GenerateBodyHash(b); err != nil {
		return err
	} else {
		bodyHash = h
	}

	sig, err := pk.Sign(util.ConcatSlice([][]byte{bodyHash.Bytes(), b}))
	if err != nil {
		return err
	}
	pb.BaseBallotV0.signer = pk.Publickey()
	pb.BaseBallotV0.signature = sig
	pb.BaseBallotV0.signedAt = localtime.Now()
	pb.bh = bodyHash

	if h, err := pb.GenerateHash(b); err != nil {
		return err
	} else {
		pb.h = h
	}

	return nil
}
