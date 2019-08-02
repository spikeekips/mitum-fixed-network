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
	ProposalType     common.DataType = common.NewDataType(3, "proposal")
	ProposalHashHint string          = "pp"
)

func NewProposalHash(b []byte) (hash.Hash, error) {
	return hash.NewDoubleSHAHash(ProposalHashHint, b)
}

func IsProposalHash(h hash.Hash) bool {
	return h.Hint() == ProposalHashHint
}

type Proposal struct {
	seal.BaseSeal
	body ProposalBody
}

func NewProposal(
	height Height,
	round Round,
	lastBlock hash.Hash,
	proposer node.Address,
	transactions []hash.Hash,
) (Proposal, error) {
	body := ProposalBody{
		height:       height,
		round:        round,
		lastBlock:    lastBlock,
		proposer:     proposer,
		transactions: transactions,
	}

	h, err := body.makeHash()
	if err != nil {
		return Proposal{}, err
	}

	body.hash = h

	return Proposal{BaseSeal: seal.NewBaseSeal(body), body: body}, nil
}

func (pp Proposal) MarshalJSON() ([]byte, error) {
	return json.Marshal(pp.BaseSeal)
}

func (pp Proposal) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, pp.BaseSeal)
}

func (pp *Proposal) DecodeRLP(s *rlp.Stream) error {
	var raw seal.RLPDecodeSeal
	if err := s.Decode(&raw); err != nil {
		return err
	}

	var body ProposalBody
	if err := rlp.DecodeBytes(raw.Body, &body); err != nil {
		return err
	}
	bsl := &seal.BaseSeal{}
	bsl = bsl.
		SetType(raw.Type).
		SetHash(raw.Hash).
		SetHeader(raw.Header).
		SetBody(body)

	pp.BaseSeal = *bsl
	pp.body = body

	if err := pp.IsValid(); err != nil {
		return err
	}

	return nil
}

func (pp Proposal) Body() seal.Body {
	return pp.body
}

func (pp Proposal) Type() common.DataType {
	return ProposalType
}

func (pp Proposal) Proposer() node.Address {
	return pp.body.proposer
}

func (pp Proposal) Height() Height {
	return pp.body.height
}

func (pp Proposal) Round() Round {
	return pp.body.round
}

func (pp Proposal) LastBlock() hash.Hash {
	return pp.body.lastBlock
}

func (pp Proposal) IsValid() error {
	if err := pp.BaseSeal.IsValid(); err != nil {
		return err
	}

	if err := pp.body.IsValid(); err != nil {
		return err
	}

	h0, err := pp.body.makeHash()
	if err != nil {
		return err
	} else if !h0.Equal(pp.body.Hash()) {
		return xerrors.Errorf("hash does not match; expected=%q hash=Tq", h0, pp.body.Hash())
	}

	return nil
}

type ProposalBody struct {
	hash         hash.Hash
	height       Height
	round        Round
	lastBlock    hash.Hash
	proposer     node.Address
	transactions []hash.Hash
}

func (ppb ProposalBody) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hash":         ppb.hash,
		"height":       ppb.height,
		"round":        ppb.round,
		"last_block":   ppb.lastBlock,
		"proposer":     ppb.proposer,
		"transactions": ppb.transactions,
	})
}

func (ppb ProposalBody) String() string {
	b, _ := common.EncodeJSON(ppb, true, false)
	return string(b)
}

func (ppb ProposalBody) Hash() hash.Hash {
	return ppb.hash
}

func (ppb ProposalBody) Type() common.DataType {
	return ProposalType
}

func (ppb ProposalBody) Height() Height {
	return ppb.height
}

func (ppb ProposalBody) Round() Round {
	return ppb.round
}

func (ppb ProposalBody) LastBlock() hash.Hash {
	return ppb.lastBlock
}

func (ppb ProposalBody) Proposer() node.Address {
	return ppb.proposer
}

func (ppb ProposalBody) IsValid() error {
	if err := ppb.hash.IsValid(); err != nil {
		return err
	} else if !IsProposalHash(ppb.hash) {
		return xerrors.Errorf("Proposal.Hash() is not valid hash; hash=%q", ppb.hash)
	}

	if err := ppb.proposer.IsValid(); err != nil {
		return err
	} else if !node.IsAddress(ppb.proposer) {
		return xerrors.Errorf("proposer is not valid node.Address; node=%q", ppb.proposer)
	}

	if err := ppb.lastBlock.IsValid(); err != nil {
		return err
	} else if !IsBlockHash(ppb.lastBlock) {
		return xerrors.Errorf("lastBlock is not block hash; lastBlock=%q", ppb.lastBlock)
	}

	return nil
}

func (ppb ProposalBody) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, struct {
		HS hash.Hash
		H  Height
		R  Round
		L  hash.Hash
		P  node.Address
		T  []hash.Hash
	}{
		HS: ppb.hash,
		H:  ppb.height,
		R:  ppb.round,
		L:  ppb.lastBlock,
		P:  ppb.proposer,
		T:  ppb.transactions,
	})
}

func (ppb *ProposalBody) DecodeRLP(s *rlp.Stream) error {
	var body struct {
		HS hash.Hash
		H  Height
		R  Round
		L  hash.Hash
		P  node.Address
		T  []hash.Hash
	}
	if err := s.Decode(&body); err != nil {
		return err
	}

	ppb.hash = body.HS
	ppb.height = body.H
	ppb.round = body.R
	ppb.lastBlock = body.L
	ppb.proposer = body.P
	ppb.transactions = body.T

	return nil
}

func (ppb ProposalBody) makeHash() (hash.Hash, error) {
	body := ProposalBody{
		height:       ppb.height,
		round:        ppb.round,
		lastBlock:    ppb.lastBlock,
		proposer:     ppb.proposer,
		transactions: ppb.transactions,
	}

	b, err := rlp.EncodeToBytes(body)
	if err != nil {
		return hash.Hash{}, err
	}

	return hash.NewDoubleSHAHash(ProposalHashHint, b)
}
