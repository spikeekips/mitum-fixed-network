package memberlist

import (
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/localtime"
)

type NodeMessagePackerJSON struct {
	Node      base.Address  `json:"node"`
	Address   string        `json:"address"`
	Publish   string        `json:"publish"`
	Insecure  bool          `json:"insecure"`
	Body      []byte        `json:"body"`
	ConID     string        `json:"connection_id,omitempty"`
	Signer    key.Publickey `json:"signer"`
	Signature key.Signature `json:"signature"`
	SignedAt  time.Time     `json:"signed_at"`
}

func (ms NodeMessage) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(NodeMessagePackerJSON{
		Node:      ms.node,
		Address:   ms.ConnInfo.Address,
		Publish:   ms.ConnInfo.URL().String(),
		Insecure:  ms.ConnInfo.Insecure(),
		Body:      ms.body,
		ConID:     ms.connid,
		Signer:    ms.signer,
		Signature: ms.signature,
		SignedAt:  ms.signedAt,
	})
}

type NodeMessageUnpackerJSON struct {
	Node      base.AddressDecoder  `json:"node"`
	Address   string               `json:"address"`
	Publish   string               `json:"publish"`
	Insecure  bool                 `json:"insecure"`
	Body      []byte               `json:"body"`
	ConID     string               `json:"connection_id,omitempty"`
	Signer    key.PublickeyDecoder `json:"signer"`
	Signature key.Signature        `json:"signature"`
	SignedAt  localtime.Time       `json:"signed_at"`
}

func (ms *NodeMessage) Unpack(b []byte, enc encoder.Encoder) error {
	var ums NodeMessageUnpackerJSON
	if err := util.JSON.Unmarshal(b, &ums); err != nil {
		return errors.Wrap(err, "failed to unmarshal NodeMessage")
	}

	publish, err := network.ParseURL(ums.Publish, false)
	if err != nil {
		return err
	}

	node, err := ums.Node.Encode(enc)
	if err != nil {
		return err
	}

	signer, err := ums.Signer.Encode(enc)
	if err != nil {
		return err
	}

	ms.node = node
	ms.ConnInfo = NewConnInfo(ums.Address, publish, ums.Insecure)
	ms.body = ums.Body
	ms.connid = ums.ConID
	ms.signer = signer
	ms.signature = ums.Signature
	ms.signedAt = ums.SignedAt.Time

	return nil
}
