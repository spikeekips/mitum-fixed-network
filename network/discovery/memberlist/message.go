package memberlist

import (
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
)

type NodeMessage struct {
	ConnInfo
	node      base.Address
	body      []byte
	connid    string
	signer    key.Publickey
	signature key.Signature
	signedAt  time.Time
}

func NewNodeMessage(
	n base.Address,
	connInfo ConnInfo,
	body []byte,
	connid string,
) NodeMessage {
	return NodeMessage{
		ConnInfo: connInfo,
		node:     n,
		body:     body,
		connid:   connid,
	}
}

func (ms NodeMessage) IsValid(networkID []byte) error {
	if err := isvalid.Check(networkID, false,
		ms.ConnInfo,
		ms.node,
		ms.signer,
		ms.signature,
	); err != nil {
		return isvalid.InvalidError.Errorf("invalid NodeMessage: %w", err)
	}

	switch u, addr, err := publishToAddress(ms.ConnInfo.URL()); {
	case err != nil:
		return err
	case ms.ConnInfo.URL().String() != u.String():
		return isvalid.InvalidError.Errorf("wrong publish url, %q", ms.ConnInfo.URL().String())
	case ms.ConnInfo.Address != addr:
		return isvalid.InvalidError.Errorf("wrong address of publish, %q != %q", ms.ConnInfo.Address, addr)
	}

	if err := ms.signer.Verify(ms.signatureBody(networkID), ms.signature); err != nil {
		return isvalid.InvalidError.Errorf("failed to verify node message: %w", err)
	}

	return nil
}

func (ms NodeMessage) signatureBody(networkID base.NetworkID) []byte {
	return util.ConcatBytesSlice(
		ms.node.Bytes(),
		ms.ConnInfo.Bytes(),
		ms.body,
		[]byte(ms.connid),
		[]byte(localtime.RFC3339(ms.signedAt)),
		networkID,
	)
}

func (ms *NodeMessage) sign(pk key.Privatekey, networkID base.NetworkID) error {
	ms.signedAt = localtime.UTCNow()

	sig, err := pk.Sign(ms.signatureBody(networkID))
	if err != nil {
		return errors.Wrap(err, "failed to sign NodeMessage")
	}

	ms.signer = pk.Publickey()
	ms.signature = sig

	return nil
}

func (ms NodeMessage) Node() base.Address {
	return ms.node
}

func (ms NodeMessage) Signer() key.Publickey {
	return ms.signer
}

func (ms NodeMessage) SignedAt() time.Time {
	return ms.signedAt
}
