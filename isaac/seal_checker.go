package isaac

import "github.com/spikeekips/mitum/seal"

// TODO SealValidationChecker should be done before ConsensusStates
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

func (svc SealValidationChecker) CheckByType() (bool, error) {
	// TODO check by seal types.
	return true, nil
}
