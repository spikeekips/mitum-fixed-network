package contrib

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/base/tree"
	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/isaac"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

var Hinters = []hint.Hinter{
	bsonenc.Encoder{},
	jsonenc.Encoder{},
	ballot.INITBallotV0{},
	ballot.ProposalV0{},
	ballot.SIGNBallotV0{},
	ballot.ACCEPTBallotV0{},
	ballot.INITBallotFactV0{},
	ballot.ProposalFactV0{},
	ballot.SIGNBallotFactV0{},
	ballot.ACCEPTBallotFactV0{},
	base.VoteproofV0{},
	base.BaseNodeV0{},
	block.BlockV0{},
	block.ManifestV0{},
	block.BlockConsensusInfoV0{},
	block.SuffrageInfoV0{},
	key.EtherPrivatekey{},
	key.EtherPublickey{},
	key.BTCPrivatekey{},
	key.BTCPublickey{},
	key.StellarPrivatekey{},
	key.StellarPublickey{},
	valuehash.SHA256{},
	valuehash.SHA512{},
	valuehash.Bytes{},
	operation.BaseSeal{},
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
	contestlib.ContestAddress(""),
}
