package ballot

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	INITFactHint   = hint.NewHint(base.INITBallotFactType, "v0.0.1")
	INITFactHinter = INITFact{BaseFact: BaseFact{BaseHinter: hint.NewBaseHinter(INITFactHint)}}
	INITHint       = hint.NewHint(base.INITBallotType, "v0.0.1")
	INITHinter     = INIT{BaseSeal: BaseSeal{BaseSeal: seal.NewBaseSealWithHint(INITHint)}}
)

type INITFact struct {
	BaseFact
	previousBlock valuehash.Hash
}

func NewINITFact(
	height base.Height,
	round base.Round,
	previousBlock valuehash.Hash,
) INITFact {
	fact := INITFact{
		BaseFact: NewBaseFact(
			INITFactHint,
			height,
			round,
		),
		previousBlock: previousBlock,
	}

	fact.BaseFact.h = valuehash.NewSHA256(fact.bytes())

	return fact
}

func (fact INITFact) IsValid([]byte) error {
	if err := isValidFact(fact); err != nil {
		return err
	}

	return isvalid.Check(nil, false, fact.previousBlock)
}

func (fact INITFact) bytes() []byte {
	var bp []byte
	if fact.previousBlock != nil {
		bp = fact.previousBlock.Bytes()
	}

	return util.ConcatBytesSlice(fact.BaseFact.bytes(), bp)
}

func (fact INITFact) PreviousBlock() valuehash.Hash {
	return fact.previousBlock
}

type INIT struct {
	BaseSeal
}

func NewINIT(
	fact INITFact,
	n base.Address,
	baseVoteproof base.Voteproof,
	acceptVoteproof base.Voteproof,
	pk key.Privatekey,
	networkID base.NetworkID,
) (INIT, error) {
	b, err := NewBaseSeal(INITHint, fact, n, baseVoteproof, acceptVoteproof, pk, networkID)
	if err != nil {
		return INIT{}, err
	}

	return INIT{BaseSeal: b}, nil
}

func (sl INIT) Fact() base.INITBallotFact {
	return sl.RawFact().(base.INITBallotFact)
}

func (sl INIT) ACCEPTVoteproof() base.Voteproof {
	if sl.acceptVoteproof != nil {
		return sl.acceptVoteproof
	}

	if sl.baseVoteproof != nil && sl.baseVoteproof.Stage() == base.StageACCEPT {
		return sl.baseVoteproof
	}

	return nil
}

func (sl INIT) IsValid(networkID []byte) error {
	if err := sl.BaseSeal.IsValid(networkID); err != nil {
		return fmt.Errorf("invalid init ballot: %w", err)
	}

	if _, ok := sl.Fact().(INITFact); !ok {
		return errors.Errorf("invalid fact of init ballot; %T", sl.Fact())
	}

	if err := sl.isValidBaseVoteproof(); err != nil {
		return isvalid.InvalidError.Wrap(err)
	}

	if err := sl.isValidACCEPTVoteproof(); err != nil {
		return isvalid.InvalidError.Wrap(err)
	}

	return nil
}

func (sl INIT) isValidBaseVoteproof() error {
	switch sl.baseVoteproof.Stage() {
	case base.StageACCEPT:
		if err := sl.isValidACCEPTBaseVoteproof(); err != nil {
			return isvalid.InvalidError.Wrap(err)
		}
	case base.StageINIT:
		if err := sl.isValidINITBaseVoteproof(); err != nil {
			return isvalid.InvalidError.Wrap(err)
		}
	}

	return nil
}

func (sl INIT) isValidACCEPTBaseVoteproof() error {
	fact := sl.Fact()
	fh := fact.Height()
	fr := fact.Round()
	vh := sl.baseVoteproof.Height()
	vr := sl.baseVoteproof.Round()

	if sl.baseVoteproof.Result() == base.VoteResultMajority {
		if sl.acceptVoteproof != nil {
			return errors.Errorf("not empty accept voteproof with accept base voteproof in init ballot")
		}

		switch {
		case fh != vh+1:
			return errors.Errorf("wrong height of init ballot + accept base voteproof; fact=%d voteproof=%d+1", fh, vh)
		case fr != base.Round(0):
			return errors.Errorf("wrong round of init ballot + accept base voteproof; fact round is not 0, %d", fr)
		default:
			return nil
		}
	}

	// NOTE DRAW
	if sl.acceptVoteproof == nil {
		return errors.Errorf("empty accept voteproof with draw accept base voteproof in init ballot")
	}

	switch {
	case fh != vh:
		return errors.Errorf("wrong height of init ballot + draw accept base voteproof; fact=%d voteproof=%d", fh, vh)
	case fr != vr+1:
		return errors.Errorf(
			"wrong round of init ballot + draw accept base voteproof; fact round=%d voteproof=%d", fr, vr)
	}

	return nil
}

func (sl INIT) isValidINITBaseVoteproof() error {
	if sl.acceptVoteproof == nil {
		return errors.Errorf("empty accept voteproof with init base voteproof")
	}

	fact := sl.Fact()
	fh := fact.Height()
	fr := fact.Round()
	vh := sl.baseVoteproof.Height()
	vr := sl.baseVoteproof.Round()

	switch {
	case fh != vh:
		return errors.Errorf("wrong height of init ballot + init base voteproof; fact=%d voteproof=%d", fh, vh)
	case fr != vr+1:
		return errors.Errorf("wrong round of init ballot + init base voteproof; fact=%d voteproof=%d+1", fr, vr)
	default:
		return nil
	}
}

func (sl INIT) isValidACCEPTVoteproof() error {
	if sl.acceptVoteproof == nil {
		return nil
	}

	fact := sl.Fact()

	if s := sl.acceptVoteproof.Stage(); s != base.StageACCEPT {
		return errors.Errorf("wrong stage of accept voteproof in init ballot; %q", s)
	}

	if r := sl.acceptVoteproof.Result(); r != base.VoteResultMajority {
		return errors.Errorf("wrong result of accept voteproof in init ballot; %q", r)
	}

	if h := sl.acceptVoteproof.Height(); h != fact.Height()-1 {
		return errors.Errorf(
			"wrong height of accept voteproof in init ballot; accept voteproof=%d == ballot=%d - 1", h, fact.Height())
	}

	return nil
}
