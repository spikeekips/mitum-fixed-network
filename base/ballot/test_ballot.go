// +build test

package ballot

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
)

func SignBaseBallotV0WithTime(blt Ballot, bb BaseBallotV0, pk key.Privatekey, networkID []byte, t time.Time) (BaseBallotV0, error) {
	if i, err := SignBaseBallotV0(blt, bb, pk, networkID); err != nil {
		return BaseBallotV0{}, err
	} else {
		i.signedAt = t

		return i, nil
	}
}

func (ib *INITBallotV0) SignWithTime(pk key.Privatekey, networkID []byte, t time.Time) error {
	if newBase, err := SignBaseBallotV0WithTime(ib, ib.BaseBallotV0, pk, networkID, t); err != nil {
		return err
	} else {
		ib.BaseBallotV0 = newBase
		ib.BaseBallotV0 = ib.BaseBallotV0.SetHash(ib.GenerateHash())

		return nil
	}
}
