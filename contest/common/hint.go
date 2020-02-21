package common

import (
	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/valuehash"
)

var Hinters = [][2]interface{}{
	{"contest-address", ContestAddress("")},
	{"encoder-bson", encoder.BSONEncoder{}},
	{"encoder-rlp", encoder.RLPEncoder{}},
	{"encoder-json", encoder.JSONEncoder{}},
	{"ballot-init", isaac.INITBallotV0{}},
	{"ballot=proposal", isaac.ProposalV0{}},
	{"ballot-sign", isaac.SIGNBallotV0{}},
	{"ballot-accept", isaac.ACCEPTBallotV0{}},
	{"ballot-init-fact", isaac.INITBallotFactV0{}},
	{"ballot-proposal-fact", isaac.ProposalFactV0{}},
	{"ballot-sign-fact", isaac.SIGNBallotFactV0{}},
	{"ballot-accept-fact", isaac.ACCEPTBallotFactV0{}},
	{"voteproof", isaac.VoteproofV0{}},
	{"block", isaac.BlockV0{}},
	{"privatekey-ether", key.EtherPrivatekey{}},
	{"publickey-ether", key.EtherPublickey{}},
	{"privatekey-btc", key.BTCPrivatekey{}},
	{"publickey-btc", key.BTCPublickey{}},
	{"privatekey-stellar", key.StellarPrivatekey{}},
	{"publickey-stellar", key.StellarPublickey{}},
	{"hash-sha256", valuehash.SHA256{}},
	{"hash-sha512", valuehash.SHA512{}},
	{"hash-dummy", valuehash.Dummy{}},
}

var HintTypes = [][2]interface{}{
	{"operation", isaac.OperationType},
	{"block-operations", isaac.BlockOperationType},
	{"block=states", isaac.BlockStatesType},
	{"block-state", isaac.BlockStateType},
}
