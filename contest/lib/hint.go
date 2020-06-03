package contestlib

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/base/tree"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/isaac"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
)

var Hinters = []hint.Hinter{
	ContestAddress(""),
	bsonencoder.Encoder{},
	jsonencoder.Encoder{},
	ballot.INITBallotV0{},
	ballot.ProposalV0{},
	ballot.SIGNBallotV0{},
	ballot.ACCEPTBallotV0{},
	ballot.INITBallotFactV0{},
	ballot.ProposalFactV0{},
	ballot.SIGNBallotFactV0{},
	ballot.ACCEPTBallotFactV0{},
	base.VoteproofV0{},
	block.BlockV0{},
	block.ManifestV0{},
	block.BlockConsensusInfoV0{},
	key.EtherPrivatekey{},
	key.EtherPublickey{},
	key.BTCPrivatekey{},
	key.BTCPublickey{},
	key.StellarPrivatekey{},
	key.StellarPublickey{},
	valuehash.SHA256{},
	valuehash.SHA512{},
	valuehash.Dummy{},
	operation.Seal{},
	tree.AVLTree{},
	operation.OperationAVLNode{},
	isaac.PolicyOperationBodyV0{},
	isaac.SetPolicyOperationV0{},
	isaac.SetPolicyOperationFactV0{},
	state.StateV0{},
	state.OperationInfoV0{},
	state.StateV0AVLNode{},
	state.BytesValue{},
	state.DurationValue{},
	state.HintedValue{},
	state.NumberValue{},
	state.SliceValue{},
	state.StringValue{},
}
