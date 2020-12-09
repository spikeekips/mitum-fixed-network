package network

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
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
		"url":        ni.u,
		"config":     ni.config,
		"suffrage":   ni.nodes,
	}))
}

type NodeInfoV0UnpackerBSON struct {
	ND  bson.Raw               `bson:"node"`
	NID base.NetworkID         `bson:"network_id"`
	ST  base.State             `bson:"state"`
	LB  bson.Raw               `bson:"last_block"`
	VS  util.Version           `bson:"version"`
	UL  string                 `bson:"url"`
	CO  map[string]interface{} `bson:"config"`
	SF  []bson.Raw             `bson:"suffrage"`
}

func (ni *NodeInfoV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var nni NodeInfoV0UnpackerBSON
	if err := enc.Unmarshal(b, &nni); err != nil {
		return err
	}

	sf := make([][]byte, len(nni.SF))
	for i := range nni.SF {
		sf[i] = nni.SF[i]
	}

	return ni.unpack(enc, nni.ND, nni.NID, nni.ST, nni.LB, nni.VS, nni.UL, nni.CO, sf)
}
