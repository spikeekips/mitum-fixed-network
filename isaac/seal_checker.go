package isaac

import (
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/isvalid"
)

type SealValidationChecker struct {
	// NOTE SealValidationChecker should be done before ConsensusStates
	seal      seal.Seal
	storage   storage.Storage
	policy    *LocalPolicy
	sealCache cache.Cache
}

func NewSealValidationChecker(
	sl seal.Seal,
	storage storage.Storage,
	policy *LocalPolicy,
	sealCache cache.Cache,
) SealValidationChecker {
	return SealValidationChecker{seal: sl, storage: storage, policy: policy, sealCache: sealCache}
}

func (svc SealValidationChecker) CheckIsValid() (bool, error) {
	if err := svc.seal.IsValid(svc.policy.NetworkID()); err != nil {
		return false, err
	}

	return true, nil
}

func (svc SealValidationChecker) CheckIsKnown() (bool, error) {
	if svc.sealCache.Has(svc.seal.Hash().String()) {
		return false, util.CheckerNilError.Errorf("seal is known")
	} else if err := svc.sealCache.Set(svc.seal.Hash().String(), struct{}{}, 0); err != nil {
		return false, util.CheckerNilError.Errorf("failed to set cache for seal: %w", err)
	}

	return true, nil
}

func (svc SealValidationChecker) IsValidOperationSeal() (bool, error) {
	var os operation.Seal
	if s, ok := svc.seal.(operation.Seal); !ok {
		return true, nil
	} else {
		os = s
	}

	if l := len(os.Operations()); l < 1 {
		return false, isvalid.InvalidError.Errorf("empty operations")
	} else if uint(l) > svc.policy.MaxOperationsInSeal() {
		return false, isvalid.InvalidError.Errorf("operations over limit; %d > %d", l, svc.policy.MaxOperationsInSeal())
	}

	return true, nil
}
