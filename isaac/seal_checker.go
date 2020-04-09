package isaac

import "github.com/spikeekips/mitum/base/seal"

// NOTE SealValidationChecker should be done before ConsensusStates

type SealValidationChecker struct {
	seal seal.Seal
	b    []byte
}

func (svc SealValidationChecker) CheckValidation() (bool, error) {
	if err := svc.seal.IsValid(svc.b); err != nil {
		return false, err
	}

	return true, nil
}
