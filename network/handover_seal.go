package network

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type HandoverSeal interface {
	seal.Seal
	Address() base.Address
	ConnInfo() ConnInfo
}

type StartHandoverSeal interface {
	HandoverSeal
}

type PingHandoverSeal interface {
	HandoverSeal
}

type EndHandoverSeal interface {
	HandoverSeal
}

type HandoverSealV0 struct {
	seal.BaseSeal
	ad base.Address
	ci ConnInfo
}

var (
	StartHandoverSealV0Type   = hint.Type("start-handover-seal")
	StartHandoverSealV0Hint   = hint.NewHint(StartHandoverSealV0Type, "v0.0.1")
	StartHandoverSealV0Hinter = HandoverSealV0{BaseSeal: seal.NewBaseSealWithHint(StartHandoverSealV0Hint)}
	PingHandoverSealV0Type    = hint.Type("ping-handover-seal")
	PingHandoverSealV0Hint    = hint.NewHint(PingHandoverSealV0Type, "v0.0.1")
	PingHandoverSealV0Hinter  = HandoverSealV0{BaseSeal: seal.NewBaseSealWithHint(PingHandoverSealV0Hint)}
	EndHandoverSealV0Type     = hint.Type("end-handover-seal")
	EndHandoverSealV0Hint     = hint.NewHint(EndHandoverSealV0Type, "v0.0.1")
	EndHandoverSealV0Hinter   = HandoverSealV0{BaseSeal: seal.NewBaseSealWithHint(EndHandoverSealV0Hint)}
)

func NewHandoverSealV0(
	ht hint.Hint,
	pk key.Privatekey,
	ad base.Address,
	ci ConnInfo,
	networkID []byte,
) (HandoverSealV0, error) {
	if ci == nil {
		return HandoverSealV0{}, fmt.Errorf("empty ConnInfo")
	}

	sl := HandoverSealV0{
		BaseSeal: seal.NewBaseSealWithHint(ht),
		ad:       ad,
		ci:       ci,
	}

	sl.GenerateBodyHashFunc = func() (valuehash.Hash, error) {
		return valuehash.NewSHA256(sl.BodyBytes()), nil
	}

	if err := sl.Sign(pk, networkID); err != nil {
		return HandoverSealV0{}, err
	}

	return sl, nil
}

func (sl HandoverSealV0) IsValid(networkID []byte) error {
	if sl.ci == nil {
		return isvalid.InvalidError.Errorf("empty ConnInfo")
	}

	if err := sl.BaseSeal.IsValid(networkID); err != nil {
		return err
	}

	return isvalid.Check([]isvalid.IsValider{sl.ad, sl.ci}, nil, false)
}

func (sl HandoverSealV0) BodyBytes() []byte {
	return util.ConcatBytesSlice(sl.BaseSeal.BodyBytes(), sl.ad.Bytes(), sl.ci.Bytes())
}

func (sl HandoverSealV0) Address() base.Address {
	return sl.ad
}

func (sl HandoverSealV0) ConnInfo() ConnInfo {
	return sl.ci
}

func IsValidHandoverSeal(local *node.Local, sl HandoverSeal, networkID base.NetworkID) error {
	if err := sl.IsValid(networkID); err != nil {
		return isvalid.InvalidError.Wrap(err)
	}

	if !sl.Address().Equal(local.Address()) {
		return isvalid.InvalidError.Errorf("handover seal not from local node")
	}

	if !sl.Signer().Equal(local.Publickey()) {
		return isvalid.InvalidError.Errorf("handover seal not signed by local")
	}

	if !localtime.WithinNow(sl.SignedAt(), time.Second*5) {
		return errors.Errorf("too old or new handover seal")
	}

	return nil
}
