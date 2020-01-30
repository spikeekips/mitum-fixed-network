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
	stage     Stage
	threshold Threshold
	result    VoteProofType
	majority  Fact
	facts     map[valuehash.Hash]Fact       // key: Fact.Hash(), value: Fact
	ballots   map[Address]valuehash.Hash    // key: node Address, value: ballot hash
	votes     map[Address]VoteProofNodeFact // key: node Address, value: VoteProofNodeFact
}

func (vr VoteProof) IsFinished() bool {
	return vr.result != VoteProofNotYet
}

func (vr VoteProof) Height() Height {
	return vr.height
}

func (vr VoteProof) Round() Round {
	return vr.round
}

func (vr VoteProof) Stage() Stage {
	return vr.stage
}

func (vr VoteProof) Result() VoteProofType {
	return vr.result
}

func (vr VoteProof) Ballots() map[Address]valuehash.Hash {
	return vr.ballots
}

func (vr VoteProof) Bytes() []byte {
	return nil
}

func (vr VoteProof) IsValid(b []byte) error {
	if err := vr.isValidFields(b); err != nil {
		return err
	}

	// check majority
	if len(vr.votes) < int(vr.threshold.Threshold) {
		if vr.result != VoteProofNotYet {
			return xerrors.Errorf("result should be not-yet: %s", vr.result)
		}

		return nil
	}

	return vr.isValidCheckMajority(b)
}

func (vr VoteProof) isValidCheckMajority(b []byte) error {
	counts := map[valuehash.Hash]uint{}
	for _, f := range vr.votes {
		counts[f.fact]++
	}

	var set []uint
	byCount := map[uint]valuehash.Hash{}
	for h, c := range counts {
		set = append(set, c)
		byCount[c] = h
	}

	var fact Fact
	var factHash valuehash.Hash
	var result VoteProofType
	switch index := FindMajority(vr.threshold.Total, vr.threshold.Threshold, set...); index {
	case -1:
		result = VoteProofNotYet
	case -2:
		result = VoteProofDraw
	default:
		result = VoteProofMajority
		factHash = byCount[set[index]]
		fact = vr.facts[factHash]
	}

	if vr.result != result {
		return xerrors.Errorf("result mismatch; vr.result=%s != result=%s", vr.result, result)
	}

	if fact == nil {
		if vr.majority != nil {
			return xerrors.Errorf("result should be nil, but not")
		}
	} else {
		mhash, err := vr.majority.Hash(b)
		if err != nil {
			return err
		}

		if !mhash.Equal(factHash) {
			return xerrors.Errorf("fact hash mismatch; vr.majority=%s != fact=%s", mhash, factHash)
		}
	}

	return nil
}

func (vr VoteProof) isValidFields(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		vr.height,
		vr.stage,
		vr.threshold,
		vr.result,
	}, b); err != nil {
		return err
	}

	if vr.majority == nil {
		if vr.result == VoteProofMajority {
			return InvalidError.Wrapf("empty majority")
		}
	} else {
		if err := vr.majority.IsValid(b); err != nil {
			return err
		}
	}

	if len(vr.facts) < 1 {
		return InvalidError.Wrapf("empty facts")
	}

	if len(vr.ballots) < 1 {
		return InvalidError.Wrapf("empty ballots")
	}

	if len(vr.votes) < 1 {
		return InvalidError.Wrapf("empty votes")
	}

	if len(vr.ballots) != len(vr.votes) {
		return InvalidError.Wrapf("vote count does not match: ballots=%d votes=%d", len(vr.ballots), len(vr.votes))
	}

	for k := range vr.ballots {
		if _, found := vr.votes[k]; !found {
			return xerrors.Errorf("unknown node found: %v", k)
		}
	}

	factHashes := map[valuehash.Hash]bool{}
	for _, f := range vr.votes {
		if _, found := vr.facts[f.fact]; !found {
			return xerrors.Errorf("missing fact found in facts: %s", f.fact.String())
		}
		factHashes[f.fact] = true
	}

	if len(factHashes) != len(vr.facts) {
		return xerrors.Errorf("unknown facts found in facts: %d", len(vr.facts)-len(factHashes))
	}

	for k, v := range vr.facts {
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

	for k, v := range vr.ballots {
		if err := isvalid.Check([]isvalid.IsValider{k, v}, b); err != nil {
			return err
		}
	}

	{
		var vs []isvalid.IsValider
		for node, f := range vr.votes {
			vs = append(vs, f, node)
		}
		if err := isvalid.Check(vs, b); err != nil {
			return err
		}
	}

	return nil
}
