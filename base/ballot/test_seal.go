//go:build test
// +build test

package ballot

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
)

func (sl *BaseSeal) SignWithTime(pk key.Privatekey, networkID []byte, t time.Time) error {
	return sl.BaseSeal.SignWithTime(pk, networkID, t)
}

func (sl *BaseSeal) SignWithFactAndTime(pk key.Privatekey, networkID []byte, t time.Time) error {
	sfs := sl.SignedFact().(base.BaseSignedBallotFact)
	fs := sfs.FactSign().(base.BaseBallotFactSign)

	fsb := fs.BaseFactSign
	fs.BaseFactSign = fsb.SetSignedAt(t)

	sl.sfs = sfs.SetFactSign(fs)

	return sl.BaseSeal.SignWithTime(pk, networkID, t)
}
