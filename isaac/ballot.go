package isaac

import (
	"encoding/json"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/node"
	"github.com/spikeekips/mitum/seal"
)

var (
	BallotType     common.DataType = common.NewDataType(1, "ballot")
	BallotHashHint string          = "ballot"
)

func NewBallotHash(b []byte) (hash.Hash, error) {
	return hash.NewDoubleSHAHash(BallotHashHint, b)
}

type Ballot struct {
	seal.BaseSeal
	body BallotBody
}

func NewBallot(body BallotBody) (Ballot, error) {
	return Ballot{
		BaseSeal: seal.NewBaseSeal(body),
		body:     body,
	}, nil
}

func (ib Ballot) MarshalJSON() ([]byte, error) {
	return json.Marshal(ib.BaseSeal)
}

func (ib Ballot) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, ib.BaseSeal)
}

func (ib *Ballot) DecodeRLP(s *rlp.Stream) error {
	var raw seal.RLPDecodeSeal
	if err := s.Decode(&raw); err != nil {
		return err
	}

	var body BallotBody
	if err := rlp.DecodeBytes(raw.Body, &body); err != nil {
		return err
	}
	bsl := &seal.BaseSeal{}
	bsl = bsl.
		SetType(raw.Type).
		SetHash(raw.Hash).
		SetHeader(raw.Header).
		SetBody(body)

	*ib = Ballot{BaseSeal: *bsl, body: body}

	if err := ib.IsValid(); err != nil {
		return err
	}

	return nil
}

func (ib Ballot) Body() seal.Body {
	return ib.body
}

func (ib Ballot) Type() common.DataType {
	return BallotType
}

func (ib Ballot) Node() node.Address {
	return ib.body.Node()
}

func (ib Ballot) Height() Height {
	return ib.body.Height()
}

func (ib Ballot) Round() Round {
	return ib.body.Round()
}

func (ib Ballot) Stage() Stage {
	return ib.body.Stage()
}

func (ib Ballot) Proposal() hash.Hash {
	return ib.body.Proposal()
}

func (ib Ballot) Block() hash.Hash {
	return ib.body.Block()
}

func (ib Ballot) LastBlock() hash.Hash {
	return ib.body.LastBlock()
}

func (ib Ballot) LastRound() Round {
	return ib.body.LastRound()
}

func (ib Ballot) IsValid() error {
	if err := ib.Stage().IsValid(); err != nil {
		return err
	}

	if err := ib.BaseSeal.IsValid(); err != nil {
		return err
	}

	if err := ib.Body().Hash().IsValid(); err != nil {
		return err
	} else if !IsBallotHash(ib.Body().Hash()) {
		return xerrors.Errorf("ballot.Body().Hash() is not valid hash; hash=%q", ib.Body().Hash())
	}

	if err := ib.Node().IsValid(); err != nil {
		return err
	} else if !node.IsAddress(ib.Node()) {
		return xerrors.Errorf("node is not node.Address; node=%q", ib.Node())
	}

	if err := ib.Proposal().IsValid(); err != nil {
		return err
	}

	if err := ib.Block().IsValid(); err != nil {
		return err
	} else if !IsBlockHash(ib.Block()) {
		return xerrors.Errorf("block is not block hash; block=%q", ib.Block())
	}

	if err := ib.LastBlock().IsValid(); err != nil {
		return err
	} else if !IsBlockHash(ib.LastBlock()) {
		return xerrors.Errorf("lastBlock is not block hash; lastBlock=%q", ib.LastBlock())
	}

	if ib.Block().Equal(ib.LastBlock()) {
		return xerrors.Errorf(
			"block should not be same with lastBlock; block=%q lastBlock=%q",
			ib.Block(),
			ib.LastBlock(),
		)
	}

	h0, err := ib.body.makeHash()
	if err != nil {
		return err
	} else if !h0.Equal(ib.Body().Hash()) {
		return xerrors.Errorf("hash does not match; expected=%q hash=Tq", h0, ib.Body().Hash())
	}

	return nil
}

func (ib Ballot) Empty() bool {
	return ib.body == nil || ib.body.Stage().IsValid() != nil
}

type BallotBody interface {
	seal.Body
	makeHash() (hash.Hash, error)
	Node() node.Address
	Stage() Stage
	Height() Height
	Round() Round
	Proposal() hash.Hash
	Block() hash.Hash
	LastBlock() hash.Hash
	LastRound() Round
}

func IsBallotHash(h hash.Hash) bool {
	return h.Hint() == BallotHashHint
}

type BaseBallotBody struct {
	hash      hash.Hash
	node      node.Address
	stage     Stage
	height    Height
	round     Round
	proposal  hash.Hash
	block     hash.Hash
	lastBlock hash.Hash
	lastRound Round
}

func (bbb BaseBallotBody) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hash":       bbb.hash,
		"node":       bbb.node,
		"stage":      bbb.stage,
		"height":     bbb.height,
		"round":      bbb.round,
		"proposal":   bbb.proposal,
		"block":      bbb.block,
		"last_block": bbb.lastBlock,
		"last_round": bbb.lastRound,
	})
}

func (bbb BaseBallotBody) String() string {
	b, _ := common.EncodeJSON(bbb, true, false) // nolint
	return string(b)
}

func (bbb BaseBallotBody) Hash() hash.Hash {
	return bbb.hash
}

func (bbb BaseBallotBody) Type() common.DataType {
	return BallotType
}

func (bbb BaseBallotBody) Node() node.Address {
	return bbb.node
}

func (bbb BaseBallotBody) Stage() Stage {
	return bbb.stage
}

func (bbb BaseBallotBody) Height() Height {
	return bbb.height
}

func (bbb BaseBallotBody) Round() Round {
	return bbb.round
}

func (bbb BaseBallotBody) Proposal() hash.Hash {
	return bbb.proposal
}

func (bbb BaseBallotBody) Block() hash.Hash {
	return bbb.block
}

func (bbb BaseBallotBody) LastBlock() hash.Hash {
	return bbb.lastBlock
}

func (bbb BaseBallotBody) LastRound() Round {
	return bbb.lastRound
}

func (bbb BaseBallotBody) IsValid() error {
	return nil
}

func (bbb BaseBallotBody) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, struct {
		HS hash.Hash
		N  node.Address
		S  Stage
		H  Height
		R  Round
		P  hash.Hash
		B  hash.Hash
		LB hash.Hash
		LR Round
	}{
		HS: bbb.hash,
		N:  bbb.node,
		S:  bbb.stage,
		H:  bbb.height,
		R:  bbb.round,
		P:  bbb.proposal,
		B:  bbb.block,
		LB: bbb.lastBlock,
		LR: bbb.lastRound,
	})
}

func (bbb *BaseBallotBody) DecodeRLP(s *rlp.Stream) error {
	var body struct {
		HS hash.Hash
		N  node.Address
		S  Stage
		H  Height
		R  Round
		P  hash.Hash
		B  hash.Hash
		LB hash.Hash
		LR Round
	}
	if err := s.Decode(&body); err != nil {
		return err
	}

	bbb.hash = body.HS
	bbb.node = body.N
	bbb.stage = body.S
	bbb.height = body.H
	bbb.round = body.R
	bbb.proposal = body.P
	bbb.block = body.B
	bbb.lastBlock = body.LB
	bbb.lastRound = body.LR

	return nil
}

func (bbb BaseBallotBody) makeHash() (hash.Hash, error) {
	ib := BaseBallotBody{
		node:      bbb.node,
		stage:     bbb.stage,
		height:    bbb.height,
		round:     bbb.round,
		proposal:  bbb.proposal,
		block:     bbb.block,
		lastBlock: bbb.lastBlock,
		lastRound: bbb.lastRound,
	}

	b, err := rlp.EncodeToBytes(ib)
	if err != nil {
		return hash.Hash{}, err
	}

	return NewBallotHash(b)
}
