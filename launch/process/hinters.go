package process

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/network"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
)

var DefaultHinters = []hint.Hinter{
	ballot.ACCEPTBallotFactV0{},
	ballot.ACCEPTBallotV0{},
	ballot.INITBallotFactV0{},
	ballot.INITBallotV0{},
	ballot.ProposalFactV0{},
	ballot.ProposalV0{},
	ballot.SIGNBallotFactV0{},
	ballot.SIGNBallotV0{},
	base.BaseNodeV0{},
	base.StringAddress(""),
	base.VoteproofV0{},
	block.BaseBlockDataMap{},
	block.BlockV0{},
	block.ConsensusInfoV0{},
	block.ManifestV0{},
	block.SuffrageInfoV0{},
	bsonenc.Encoder{},
	jsonenc.Encoder{},
	key.BTCPrivatekeyHinter,
	key.BTCPublickeyHinter,
	key.EtherPrivatekeyHinter,
	key.EtherPublickeyHinter,
	key.StellarPrivatekeyHinter,
	key.StellarPublickeyHinter,
	network.NodeInfoV0{},
	operation.BaseFactSign{},
	operation.BaseSeal{},
	operation.FixedTreeNode{},
	operation.BaseReasonError{},
	state.BytesValue{},
	state.DurationValue{},
	state.HintedValue{},
	state.NumberValue{},
	state.SliceValue{},
	state.StateV0{},
	state.StringValue{},
	tree.BaseFixedTreeNode{},
	tree.FixedTree{},
	valuehash.Bytes{},
	valuehash.SHA256{},
	valuehash.SHA512{},
}
