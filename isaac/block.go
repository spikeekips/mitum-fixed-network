package isaac

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/hash"
)

var (
	BlockHashHint string = "bk"
)

func NewBlockHash(b []byte) (hash.Hash, error) {
	return hash.NewDoubleSHAHash(BlockHashHint, b)
}

func IsBlockHash(h hash.Hash) bool {
	return h.Hint() == BlockHashHint
}

type Block struct {
	hash      hash.Hash
	height    Height
	proposal  hash.Hash
	round     Round
	createdAt common.Time
}

func NewBlock(height Height, round Round, proposal hash.Hash) (Block, error) {
	bk := Block{
		height:    height,
		proposal:  proposal,
		round:     round,
		createdAt: common.Now(),
	}

	h, err := bk.makeHash()
	if err != nil {
		return Block{}, err
	}
	bk.hash = h

	return bk, nil
}

func NewBlockWithHash(height Height, round Round, proposal hash.Hash, newHash hash.Hash) (Block, error) {
	return Block{
		hash:      newHash,
		height:    height,
		proposal:  proposal,
		round:     round,
		createdAt: common.Now(),
	}, nil
}

func NewBlockFromVoteResult(vr VoteResult) (Block, error) {
	// TODO fix; it's just for testing
	height, ok := vr.Height().SubOK(1)
	if !ok {
		return Block{}, xerrors.Errorf("height of new block is under 0")
	}

	block, err := NewBlock(height, vr.LastRound(), vr.Proposal())
	if err != nil {
		return Block{}, err
	}

	if !block.Hash().Equal(vr.Block()) {
		return Block{}, xerrors.Errorf("new block from VoteResult does not match")
	}

	return block, nil
}

func (bk Block) makeHash() (hash.Hash, error) {
	if err := bk.proposal.IsValid(); err != nil {
		return hash.Hash{}, err
	}

	b, err := rlp.EncodeToBytes([]interface{}{
		bk.height,
		bk.round,
		bk.proposal,
	})

	if err != nil {
		return hash.Hash{}, err
	}

	return NewBlockHash(b)
}

func (bk Block) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hash":      bk.hash,
		"height":    bk.height,
		"round":     bk.round,
		"proposal":  bk.proposal,
		"createdAt": bk.createdAt,
	})
}

func (bk Block) MarshalZerologObject(e *zerolog.Event) {
	e.Object("hash", bk.hash)
	e.Str("height", bk.height.String())
	e.Uint64("round", bk.round.Uint64())
	e.Object("proposal", bk.proposal)
	e.Time("createdAt", bk.createdAt.Time)
}

func (bk Block) String() string {
	b, _ := json.Marshal(bk) // nolint
	return string(b)
}

func (bk Block) Hash() hash.Hash {
	return bk.hash
}

func (bk Block) Height() Height {
	return bk.height
}

func (bk Block) Round() Round {
	return bk.round
}

func (bk Block) Proposal() hash.Hash {
	return bk.proposal
}

func (bk Block) Equal(n Block) bool {
	if !bk.Height().Equal(n.Height()) {
		return false
	}

	if bk.Round() != n.Round() {
		return false
	}

	if !bk.Hash().Equal(n.Hash()) {
		return false
	}

	if !bk.Proposal().Equal(n.Proposal()) {
		return false
	}

	return true
}

func (bk Block) Empty() bool {
	return bk.hash.Empty()
}
