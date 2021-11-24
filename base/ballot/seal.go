package ballot

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

type BaseSeal struct {
	seal.BaseSeal
	sfs             base.SignedBallotFact
	baseVoteproof   base.Voteproof
	acceptVoteproof base.Voteproof
}

func NewBaseSeal(
	ht hint.Hint,
	fact base.BallotFact,
	n base.Address,
	baseVoteproof base.Voteproof,
	acceptVoteproof base.Voteproof,
	pk key.Privatekey,
	networkID base.NetworkID,
) (BaseSeal, error) {
	if err := fact.IsValid(nil); err != nil {
		return BaseSeal{}, err
	}

	sfs, err := base.NewBaseSignedBallotFactFromFact(fact, n, pk, networkID)
	if err != nil {
		return BaseSeal{}, err
	}

	sl := BaseSeal{
		BaseSeal:        seal.NewBaseSealWithHint(ht),
		sfs:             sfs,
		baseVoteproof:   baseVoteproof,
		acceptVoteproof: acceptVoteproof,
	}

	sl.BaseSeal.GenerateBodyHashFunc = func() (valuehash.Hash, error) {
		return valuehash.NewSHA256(sl.BodyBytes()), nil
	}

	if err := sl.BaseSeal.Sign(pk, networkID); err != nil {
		return BaseSeal{}, err
	}

	return sl, nil
}

func (sl BaseSeal) IsValid(networkID []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		sl.sfs,
		sl.BaseSeal,
	}, networkID, false); err != nil {
		return err
	}

	if err := isvalid.CheckFunc([]func() error{
		sl.isValidHasFact,
		sl.isValidSignedAt,
	}); err != nil {
		return err
	}

	if err := isvalid.Check([]isvalid.IsValider{
		sl.baseVoteproof,
		sl.acceptVoteproof,
	}, networkID, true); err != nil {
		return err
	}

	if sl.baseVoteproof != nil && !sl.baseVoteproof.IsFinished() {
		return isvalid.InvalidError.Errorf("not yet finished base voteproof found in ballot")
	}
	if sl.acceptVoteproof != nil && !sl.acceptVoteproof.IsFinished() {
		return isvalid.InvalidError.Errorf("not yet finished accept voteproof found in ballot")
	}

	return nil
}

func (sl BaseSeal) BodyBytes() []byte {
	var bsfs, bb, ba []byte
	if sl.sfs != nil {
		bsfs = sl.sfs.Bytes()
	}

	if sl.baseVoteproof != nil {
		bb = sl.baseVoteproof.Bytes()
	}

	if sl.acceptVoteproof != nil {
		ba = sl.acceptVoteproof.Bytes()
	}

	return util.ConcatBytesSlice(sl.BaseSeal.BodyBytes(), bsfs, bb, ba)
}

func (sl BaseSeal) MarshalZerologObject(e *zerolog.Event) {
	e.
		Stringer("hash", sl.Hash()).
		Stringer("hint", sl.Hint()).
		Dict("fact", marshalZerologFact(sl.sfs.Fact())).
		Dict("fact_sign", base.MarshalZerologFactSign(sl.sfs.FactSign())).
		Object("base_voteproof", sl.baseVoteproof).
		Object("accept_voteproof", sl.acceptVoteproof)
}

func (sl BaseSeal) RawFact() base.BallotFact {
	return sl.sfs.Fact()
}

func (sl BaseSeal) FactSign() base.BallotFactSign {
	return sl.sfs.FactSign()
}

func (sl BaseSeal) SignedFact() base.SignedBallotFact {
	return sl.sfs
}

func (sl BaseSeal) BaseVoteproof() base.Voteproof {
	return sl.baseVoteproof
}

func (sl BaseSeal) ACCEPTVoteproof() base.Voteproof {
	return sl.acceptVoteproof
}

func (sl *BaseSeal) SignWithFact(n base.Address, priv key.Privatekey, networkID []byte) error {
	sfs, err := base.NewBaseSignedBallotFactFromFact(sl.sfs.Fact(), n, priv, networkID)
	if err != nil {
		return err
	}

	sl.sfs = sfs

	return sl.BaseSeal.Sign(priv, networkID)
}

func (sl BaseSeal) isValidBaseVoteproofAfterINIT() error {
	switch {
	case sl.acceptVoteproof != nil:
		return errors.Errorf("not empty accept voteproof with base voteproof in proposal")
	case sl.baseVoteproof.Stage() != base.StageINIT:
		return errors.Errorf(
			"invalid base voteproof stage in proposal; should be init, not %v", sl.baseVoteproof.Stage())
	case sl.baseVoteproof.Result() != base.VoteResultMajority:
		return errors.Errorf("not majority result of base voteproof in proposal")
	case sl.baseVoteproof.Height() != sl.sfs.Fact().Height():
		return errors.Errorf(
			"wrong height of base voteproof in proposal; voteproof=%d == fact=%d",
			sl.baseVoteproof.Height(), sl.sfs.Fact().Height())
	case sl.baseVoteproof.Round() != sl.sfs.Fact().Round():
		return errors.Errorf(
			"wrong round of base voteproof in proposal; voteproof=%d == fact=%d",
			sl.baseVoteproof.Round(), sl.sfs.Fact().Round())
	}

	return nil
}

func (sl BaseSeal) isValidHasFact() error {
	var expected hint.Type
	switch sl.sfs.Fact().Stage() {
	case base.StageINIT:
		expected = base.INITBallotType
	case base.StageProposal:
		expected = base.ProposalType
	case base.StageACCEPT:
		expected = base.ACCEPTBallotType
	default:
		return errors.Errorf("invalid ballot stage found, %q", sl.sfs.Fact().Stage())
	}

	if sl.Hint().Type() != expected {
		return errors.Errorf("ballot has weird fact; %q in %q", sl.sfs.Fact().Hint().Type(), sl.Hint().Type())
	}

	return nil
}

func (sl BaseSeal) isValidSignedAt() error {
	if sl.sfs.Fact().Stage() == base.StageProposal {
		return nil
	}

	if sl.SignedAt().Before(sl.sfs.FactSign().SignedAt()) {
		return errors.Errorf("ballot is signed at before fact; %q < %q", sl.SignedAt(), sl.sfs.FactSign().SignedAt())
	}

	return nil
}
