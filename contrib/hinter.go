package contrib

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/policy"
	"github.com/spikeekips/mitum/base/state"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

var Hinters = []hint.Hinter{
	ballot.ACCEPTBallotFactV0{},
	ballot.ACCEPTBallotV0{},
	ballot.INITBallotFactV0{},
	ballot.INITBallotV0{},
	ballot.ProposalFactV0{},
	ballot.ProposalV0{},
	ballot.SIGNBallotFactV0{},
	ballot.SIGNBallotV0{},
	base.BaseNodeV0{},
	base.VoteproofV0{},
	block.ConsensusInfoV0{},
	block.BlockV0{},
	block.ManifestV0{},
	block.SuffrageInfoV0{},
	bsonenc.Encoder{},
	base.StringAddress(""),
	jsonenc.Encoder{},
	key.BTCPrivatekeyHinter,
	key.BTCPublickeyHinter,
	key.EtherPrivatekeyHinter,
	key.EtherPublickeyHinter,
	key.StellarPrivatekeyHinter,
	key.StellarPublickeyHinter,
	operation.BaseSeal{},
	state.BytesValue{},
	state.DurationValue{},
	state.HintedValue{},
	state.NumberValue{},
	state.SliceValue{},
	state.StateV0{},
	state.StringValue{},
	valuehash.Bytes{},
	valuehash.SHA256{},
	valuehash.SHA512{},
	policy.PolicyV0{},
	policy.SetPolicyFactV0{},
	policy.SetPolicyV0{},
}
