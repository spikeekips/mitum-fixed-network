package contrib

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/base/tree"
	"github.com/spikeekips/mitum/isaac"
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
	block.BlockConsensusInfoV0{},
	block.BlockV0{},
	block.ManifestV0{},
	block.SuffrageInfoV0{},
	bsonenc.Encoder{},
	base.StringAddress(""),
	isaac.PolicyOperationBodyV0{},
	isaac.SetPolicyOperationFactV0{},
	isaac.SetPolicyOperationV0{},
	jsonenc.Encoder{},
	key.BTCPrivatekey{},
	key.BTCPublickey{},
	key.EtherPrivatekey{},
	key.EtherPublickey{},
	key.StellarPrivatekey{},
	key.StellarPublickey{},
	operation.BaseSeal{},
	operation.OperationAVLNode{},
	state.BytesValue{},
	state.DurationValue{},
	state.HintedValue{},
	state.NumberValue{},
	state.SliceValue{},
	state.StateV0AVLNode{},
	state.StateV0{},
	state.StringValue{},
	tree.AVLTree{},
	valuehash.Bytes{},
	valuehash.SHA256{},
	valuehash.SHA512{},
}
