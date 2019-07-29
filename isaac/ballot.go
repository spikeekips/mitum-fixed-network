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

func (ib Ballot) IsValid() error {
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
	} else if !IsProposalHash(ib.Proposal()) {
		return xerrors.Errorf("proposal is not proposal hash; proposal=%q", ib.Proposal())
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

	if ib.Round() < 1 {
		return xerrors.Errorf("round should be greater than 0; round=%q", ib.Round())
	}

	h0, err := ib.body.makeHash()
	if err != nil {
		return err
	} else if !h0.Equal(ib.Body().Hash()) {
		return xerrors.Errorf("hash does not match; expected=%q hash=Tq", h0, ib.Body().Hash())
	}

	return nil
}

type BallotBody interface {
	seal.Body
	makeHash() (hash.Hash, error)
	Node() node.Address
	Height() Height
	Round() Round
	Stage() Stage
	Proposal() hash.Hash
	Block() hash.Hash
	LastBlock() hash.Hash
}

func IsBallotHash(h hash.Hash) bool {
	return h.Hint() == BallotHashHint
}

type BaseBallotBody struct {
	hash      hash.Hash
	node      node.Address
	height    Height
	round     Round
	proposal  hash.Hash
	block     hash.Hash
	lastBlock hash.Hash
	stage     Stage
}

func (bbb BaseBallotBody) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hash":           bbb.hash,
		"node":           bbb.node,
		"height":         bbb.height,
		"round":          bbb.round,
		"proposal":       bbb.proposal,
		"block":          bbb.block,
		"previous_block": bbb.lastBlock,
		"stage":          bbb.stage,
	})
}

func (bbb BaseBallotBody) String() string {
	b, _ := common.EncodeJSON(bbb, true, false)
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

func (bbb BaseBallotBody) Height() Height {
	return bbb.height
}

func (bbb BaseBallotBody) Round() Round {
	return bbb.round
}

func (bbb BaseBallotBody) Stage() Stage {
	return bbb.stage
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

func (bbb BaseBallotBody) IsValid() error {
	return nil
}

func (bbb BaseBallotBody) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, struct {
		HS hash.Hash
		N  node.Address
		H  Height
		R  Round
		P  hash.Hash
		B  hash.Hash
		PR hash.Hash
	}{
		HS: bbb.hash,
		N:  bbb.node,
		H:  bbb.height,
		R:  bbb.round,
		P:  bbb.proposal,
		B:  bbb.block,
		PR: bbb.lastBlock,
	})
}

func (bbb *BaseBallotBody) DecodeRLP(s *rlp.Stream) error {
	var body struct {
		HS hash.Hash
		N  node.Address
		H  Height
		R  Round
		P  hash.Hash
		B  hash.Hash
		PR hash.Hash
	}
	if err := s.Decode(&body); err != nil {
		return err
	}

	bbb.hash = body.HS
	bbb.node = body.N
	bbb.height = body.H
	bbb.round = body.R
	bbb.proposal = body.P
	bbb.block = body.B
	bbb.lastBlock = body.PR

	return nil
}

func (bbb BaseBallotBody) makeHash() (hash.Hash, error) {
	ib := BaseBallotBody{
		node:      bbb.node,
		height:    bbb.height,
		round:     bbb.round,
		proposal:  bbb.proposal,
		block:     bbb.block,
		lastBlock: bbb.lastBlock,
	}

	b, err := rlp.EncodeToBytes(ib)
	if err != nil {
		return hash.Hash{}, err
	}
	h, err := hash.NewDoubleSHAHash(BallotHashHint, b)
	if err != nil {
		return hash.Hash{}, err
	}

	return h, nil
}
