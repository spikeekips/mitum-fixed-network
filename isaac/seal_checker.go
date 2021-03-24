package isaac

import (
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/isvalid"
	"golang.org/x/xerrors"
)

type SealChecker struct {
	// NOTE SealValidationChecker should be done before ConsensusStates
	seal      seal.Seal
	database  storage.Database
	policy    *LocalPolicy
	sealCache cache.Cache
}

func NewSealChecker(
	sl seal.Seal,
	database storage.Database,
	policy *LocalPolicy,
	sealCache cache.Cache,
) SealChecker {
	return SealChecker{seal: sl, database: database, policy: policy, sealCache: sealCache}
}

func (svc SealChecker) IsValid() (bool, error) {
	if err := svc.seal.IsValid(svc.policy.NetworkID()); err != nil {
		return false, err
	}

	return true, nil
}

func (svc SealChecker) IsKnown() (bool, error) {
	if svc.sealCache == nil {
		return true, nil
	}

	cachekey := svc.seal.Hash().String()
	if svc.sealCache.Has(cachekey) {
		return false, util.IgnoreError.Errorf("seal is known")
	} else if err := svc.sealCache.Set(cachekey, struct{}{}, 0); err != nil {
		return false, util.IgnoreError.Errorf("failed to set cache for seal: %w", err)
	} else {
		return true, nil
	}
}

func (svc SealChecker) IsValidOperationSeal() (bool, error) {
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

	var notFound bool
	for i := range os.Operations() {
		if found, err := svc.database.HasOperationFact(os.Operations()[i].Fact().Hash()); err != nil {
			return false, xerrors.Errorf("failed to check HasOperationFact: %w", err)
		} else if !found {
			notFound = true

			break
		}
	}

	if !notFound {
		return false, util.IgnoreError.Errorf("operation seal does not have new operations")
	}

	return true, nil
}
