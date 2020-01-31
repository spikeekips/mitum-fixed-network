package mitum

import (
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/valuehash"
	"golang.org/x/xerrors"
)

type VoteProofType uint8

const (
	VoteProofNotYet VoteProofType = iota
	VoteProofDraw
	VoteProofMajority
)

func (vrt VoteProofType) String() string {
	switch vrt {
	case VoteProofNotYet:
		return "NOT-YET"
	case VoteProofDraw:
		return "DRAW"
	case VoteProofMajority:
		return "MAJORITY"
	default:
		return "<unknown VoteProofType>"
	}
}

func (vrt VoteProofType) IsValid([]byte) error {
	switch vrt {
	case VoteProofNotYet, VoteProofDraw, VoteProofMajority:
		return nil
	}

	return InvalidError.Wrapf("VoteProofType=%d", vrt)
}

func (vrt VoteProofType) MarshalText() ([]byte, error) {
	return []byte(vrt.String()), nil
}

type VoteProofNodeFact struct {
	fact          valuehash.Hash
	factSignature key.Signature
	signer        key.Publickey
}

func (vf VoteProofNodeFact) IsValid(b []byte) error {
	// TODO check,
	// - signer is valid Ballot.Signer()?
	if err := isvalid.Check([]isvalid.IsValider{
		vf.fact,
		vf.factSignature,
		vf.signer,
	}, b); err != nil {
		return err
	}

	return vf.signer.Verify(vf.fact.Bytes(), vf.factSignature)
}

type VoteProof struct {
	height    Height
	round     Round
	threshold Threshold
	result    VoteProofType
	stage     Stage
	majority  Fact
	facts     map[valuehash.Hash]Fact       // key: Fact.Hash(), value: Fact
	ballots   map[Address]valuehash.Hash    // key: node Address, value: ballot hash
	votes     map[Address]VoteProofNodeFact // key: node Address, value: VoteProofNodeFact
}

func (vp VoteProof) IsFinished() bool {
	return vp.result != VoteProofNotYet
}

func (vp VoteProof) Height() Height {
	return vp.height
}

func (vp VoteProof) Round() Round {
	return vp.round
}

func (vp VoteProof) Stage() Stage {
	return vp.stage
}

func (vp VoteProof) Result() VoteProofType {
	return vp.result
}

func (vp VoteProof) Ballots() map[Address]valuehash.Hash {
	return vp.ballots
}

func (vp VoteProof) Bytes() []byte {
	return nil
}

func (vp VoteProof) IsValid(b []byte) error {
	if err := vp.isValidFields(b); err != nil {
		return err
	}

	if err := vp.isValidFacts(b); err != nil {
		return err
	}

	// check majority
	if len(vp.votes) < int(vp.threshold.Threshold) {
		if vp.result != VoteProofNotYet {
			return xerrors.Errorf("result should be not-yet: %s", vp.result)
		}

		return nil
	}

	return vp.isValidCheckMajority(b)
}

func (vp VoteProof) isValidCheckMajority(b []byte) error {
	counts := map[valuehash.Hash]uint{}
	for _, f := range vp.votes { // nolint
		counts[f.fact]++
	}

	set := make([]uint, len(counts))
	byCount := map[uint]valuehash.Hash{}
	for h, c := range counts {
		set = append(set, c)
		byCount[c] = h
	}

	var fact Fact
	var factHash valuehash.Hash
	var result VoteProofType
	switch index := FindMajority(vp.threshold.Total, vp.threshold.Threshold, set...); index {
	case -1:
		result = VoteProofNotYet
	case -2:
		result = VoteProofDraw
	default:
		result = VoteProofMajority
		factHash = byCount[set[index]]
		fact = vp.facts[factHash]
	}

	if vp.result != result {
		return xerrors.Errorf("result mismatch; vp.result=%s != result=%s", vp.result, result)
	}

	if fact == nil {
		if vp.majority != nil {
			return xerrors.Errorf("result should be nil, but not")
		}
	} else {
		mhash, err := vp.majority.Hash(b)
		if err != nil {
			return err
		}

		if !mhash.Equal(factHash) {
			return xerrors.Errorf("fact hash mismatch; vp.majority=%s != fact=%s", mhash, factHash)
		}
	}

	return nil
}

func (vp VoteProof) isValidFields(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		vp.height,
		vp.stage,
		vp.threshold,
		vp.result,
	}, b); err != nil {
		return err
	}

	if vp.result != VoteProofMajority && vp.result != VoteProofDraw {
		return InvalidError.Wrapf("invalid result; result=%v", vp.result)
	}

	if vp.majority == nil {
		if vp.result != VoteProofDraw {
			return InvalidError.Wrapf("empty majority, but result is not draw; result=%v", vp.result)
		}
	} else if err := vp.majority.IsValid(b); err != nil {
		return err
	}

	if len(vp.facts) < 1 {
		return InvalidError.Wrapf("empty facts")
	}

	if len(vp.ballots) < 1 {
		return InvalidError.Wrapf("empty ballots")
	}

	if len(vp.votes) < 1 {
		return InvalidError.Wrapf("empty votes")
	}

	if len(vp.ballots) != len(vp.votes) {
		return InvalidError.Wrapf("vote count does not match: ballots=%d votes=%d", len(vp.ballots), len(vp.votes))
	}

	for k := range vp.ballots {
		if _, found := vp.votes[k]; !found {
			return xerrors.Errorf("unknown node found: %v", k)
		}
	}

	return nil
}

func (vp VoteProof) isValidFacts(b []byte) error {
	factHashes := map[valuehash.Hash]bool{}
	for node, f := range vp.votes { // nolint
		if err := node.IsValid(b); err != nil {
			return err
		}

		if _, found := vp.facts[f.fact]; !found {
			return xerrors.Errorf("missing fact found in facts: %s", f.fact.String())
		}
		factHashes[f.fact] = true
	}

	if len(factHashes) != len(vp.facts) {
		return xerrors.Errorf("unknown facts found in facts: %d", len(vp.facts)-len(factHashes))
	}

	for k, v := range vp.facts {
		if err := isvalid.Check([]isvalid.IsValider{k, v}, b); err != nil {
			return err
		}
		if h, err := v.Hash(b); err != nil {
			return err
		} else if !h.Equal(k) {
			return xerrors.Errorf(
				"factHash and Fact.Hash() does not match: factHash=%v != Fact.Hash()=%v",
				k.String(), h.String(),
			)
		}
	}

	for k, v := range vp.ballots {
		if err := isvalid.Check([]isvalid.IsValider{k, v}, b); err != nil {
			return err
		}
	}

	return nil
}
