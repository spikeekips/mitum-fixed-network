package base

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
)

var (
	BallotFactSignType   = hint.Type("ballot-fact-sign")
	BallotFactSignHint   = hint.NewHint(BallotFactSignType, "v0.0.1")
	BallotFactSignHinter = BaseBallotFactSign{BaseFactSign: BaseFactSign{
		BaseHinter: hint.NewBaseHinter(BallotFactSignHint),
	}}
	SignedBallotFactType   = hint.Type("signed-ballot-fact")
	SignedBallotFactHint   = hint.NewHint(SignedBallotFactType, "v0.0.1")
	SignedBallotFactHinter = BaseSignedBallotFact{BaseHinter: hint.NewBaseHinter(SignedBallotFactHint)}
)

type SignedBallotFact interface {
	hint.Hinter
	isvalid.IsValider
	util.Byter
	Fact() BallotFact
	FactSign() BallotFactSign
}

type BaseBallotFactSign struct {
	BaseFactSign
	node Address
}

func NewBaseBallotFactSign(n Address, pub key.Publickey, signedAt time.Time, sig key.Signature) BaseBallotFactSign {
	return BaseBallotFactSign{
		BaseFactSign: NewBaseFactSignWithHint(BallotFactSignHint, pub, sig).SetSignedAt(signedAt),
		node:         n,
	}
}

func NewBaseBallotFactSignFromFact(
	fact BallotFact,
	n Address,
	priv key.Privatekey,
	networkID NetworkID,
) (BaseBallotFactSign, error) {
	signedAt := localtime.UTCNow()
	sig, err := priv.Sign(util.ConcatBytesSlice(
		fact.Hash().Bytes(),
		localtime.NewTime(signedAt).Bytes(),
		networkID,
	))
	if err != nil {
		return BaseBallotFactSign{}, err
	}

	return NewBaseBallotFactSign(n, priv.Publickey(), signedAt, sig), nil
}

func (fs BaseBallotFactSign) IsValid([]byte) error {
	if err := isvalid.Check(
		[]isvalid.IsValider{
			fs.node,
			fs.BaseFactSign,
		}, nil, false); err != nil {
		return isvalid.InvalidError.Errorf("invalid ballot fact sign: %w", err)
	}

	return nil
}

func (fs BaseBallotFactSign) Bytes() []byte {
	return util.ConcatBytesSlice(fs.BaseFactSign.Bytes(), fs.node.Bytes())
}

func (fs BaseBallotFactSign) Node() Address {
	return fs.node
}

func IsValidBallotFactSign(fact BallotFact, fs BallotFactSign, b []byte) error {
	if fact == nil || fact.Hash() == nil {
		return isvalid.InvalidError.Errorf("empty Fact")
	}
	if fs == nil {
		return isvalid.InvalidError.Errorf("empty FactSign")
	}
	if fs.Signer() == nil {
		return isvalid.InvalidError.Errorf("FactSign has empty Signer()")
	}
	if fs.Signature() == nil {
		return isvalid.InvalidError.Errorf("FactSign has empty Signature()")
	}
	if fs.SignedAt().IsZero() {
		return isvalid.InvalidError.Errorf("FactSign has empty SignedAt()")
	}

	if fact.Stage() == StageProposal {
		switch i, ok := fact.(ProposalFact); {
		case !ok:
			return isvalid.InvalidError.Errorf("proposal fact has wrong type; %T", fact)
		case !i.Proposer().Equal(fs.Node()):
			return isvalid.InvalidError.Errorf("proposal fact is not signed by factsign node")
		}
	}

	return fs.Signer().Verify(
		util.ConcatBytesSlice(
			fact.Hash().Bytes(),
			localtime.NewTime(fs.SignedAt()).Bytes(),
			b,
		),
		fs.Signature(),
	)
}

type BaseSignedBallotFact struct {
	hint.BaseHinter
	fact     BallotFact
	factSign BallotFactSign
}

func NewBaseSignedBallotFact(fact BallotFact, factSign BallotFactSign) BaseSignedBallotFact {
	return BaseSignedBallotFact{
		BaseHinter: hint.NewBaseHinter(SignedBallotFactHint),
		fact:       fact,
		factSign:   factSign,
	}
}

func NewBaseSignedBallotFactFromFact(
	fact BallotFact,
	n Address,
	priv key.Privatekey,
	networkID NetworkID,
) (BaseSignedBallotFact, error) {
	fs, err := NewBaseBallotFactSignFromFact(fact, n, priv, networkID)
	if err != nil {
		return BaseSignedBallotFact{}, err
	}

	return NewBaseSignedBallotFact(fact, fs), nil
}

func (sfs BaseSignedBallotFact) IsValid(networkID []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		sfs.BaseHinter,
		sfs.fact,
		sfs.factSign,
	}, nil, false); err != nil {
		return err
	}

	return IsValidBallotFactSign(sfs.fact, sfs.factSign, networkID)
}

func (sfs BaseSignedBallotFact) Bytes() []byte {
	return util.ConcatBytesSlice(
		sfs.fact.Hash().Bytes(),
		sfs.factSign.Bytes(),
	)
}

func (sfs BaseSignedBallotFact) Fact() BallotFact {
	return sfs.fact
}

func (sfs BaseSignedBallotFact) FactSign() BallotFactSign {
	return sfs.factSign
}
