package isaac

import "github.com/spikeekips/mitum/base/seal"

// NOTE SealValidationChecker should be done before ConsensusStates

type SealValidationChecker struct {
	seal seal.Seal
	b    []byte
}

func NewSealValidationChecker(sl seal.Seal, b []byte) SealValidationChecker {
	return SealValidationChecker{seal: sl, b: b}
}

func (svc SealValidationChecker) CheckIsValid() (bool, error) {
	if err := svc.seal.IsValid(svc.b); err != nil {
		return false, err
	}

	return true, nil
}
