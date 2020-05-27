package isaac

import (
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

// NOTE SealValidationChecker should be done before ConsensusStates

type SealValidationChecker struct {
	seal    seal.Seal
	b       []byte
	storage storage.Storage
}

func NewSealValidationChecker(sl seal.Seal, b []byte, storage storage.Storage) SealValidationChecker {
	return SealValidationChecker{seal: sl, b: b, storage: storage}
}

func (svc SealValidationChecker) CheckIsValid() (bool, error) {
	if err := svc.seal.IsValid(svc.b); err != nil {
		return false, err
	}

	return true, nil
}

func (svc SealValidationChecker) CheckIsKnown() (bool, error) {
	if found, err := svc.storage.HasSeal(svc.seal.Hash()); err != nil {
		return false, err
	} else if found {
		return false, util.CheckerNilError.Errorf("seal is known")
	}

	return true, nil
}
