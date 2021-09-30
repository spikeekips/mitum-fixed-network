package network

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func (ni NodeInfoV0) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(bsonenc.NewHintedDoc(ni.Hint()), bson.M{
		"node":       ni.node,
		"network_id": ni.networkID,
		"state":      ni.state,
		"last_block": ni.lastBlock,
		"version":    ni.version,
		"policy":     ni.policy,
		"suffrage":   ni.nodes,
		"conninfo":   ni.ci,
	}))
}

type NodeInfoV0UnpackerBSON struct {
	ND  bson.Raw               `bson:"node"`
	NID base.NetworkID         `bson:"network_id"`
	ST  base.State             `bson:"state"`
	LB  bson.Raw               `bson:"last_block"`
	VS  util.Version           `bson:"version"`
	PO  map[string]interface{} `bson:"policy"`
	SF  []bson.Raw             `bson:"suffrage"`
	CI  bson.Raw               `bson:"conninfo"`
}

func (ni *NodeInfoV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var nni NodeInfoV0UnpackerBSON
	if err := enc.Unmarshal(b, &nni); err != nil {
		return err
	}

	sf := make([]RemoteNode, len(nni.SF))
	for i := range nni.SF {
		var r RemoteNode
		if err := r.unpackBSON(nni.SF[i], enc); err != nil {
			return err
		}

		sf[i] = r
	}

	return ni.unpack(enc, nni.ND, nni.NID, nni.ST, nni.LB, nni.VS, nni.PO, sf, nni.CI)
}

func (no RemoteNode) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(map[string]interface{}{
		"address":   no.Address,
		"publickey": no.Publickey,
		"conninfo":  no.ci,
	})
}

type RemoteNodeUnpackBSON struct {
	A  base.AddressDecoder  `bson:"address"`
	P  key.PublickeyDecoder `bson:"publickey"`
	CI bson.Raw             `bson:"conninfo"`
}

func (no *RemoteNode) unpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uno RemoteNodeUnpackBSON
	if err := bson.Unmarshal(b, &uno); err != nil {
		return err
	}

	return no.unpack(enc, uno.A, uno.P, uno.CI)
}
