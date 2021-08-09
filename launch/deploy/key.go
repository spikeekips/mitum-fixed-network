package deploy

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
)

var DatabaseInfoDeployKeysKey = "deploy_keys"

type DeployKey struct {
	k       string
	addedAt time.Time
}

func NewDeployKey() DeployKey {
	return DeployKey{
		k:       "d-" + util.UUID().String(),
		addedAt: localtime.UTCNow(),
	}
}

func (dk DeployKey) Key() string {
	return dk.k
}

func (dk DeployKey) AddedAt() time.Time {
	return dk.addedAt
}

type DeployKeyStorage struct {
	sync.RWMutex
	db   storage.Database
	keys map[string]DeployKey
}

func NewDeployKeyStorage(db storage.Database) (*DeployKeyStorage, error) {
	keys := map[string]DeployKey{}
	if db != nil {
		if i, err := loadDeployKeys(db); err != nil {
			return nil, err
		} else if i != nil {
			keys = i
		}
	}

	return &DeployKeyStorage{
		db:   db,
		keys: keys,
	}, nil
}

func (ks *DeployKeyStorage) Exists(k string) bool {
	ks.RLock()
	defer ks.RUnlock()

	_, found := ks.keys[k]

	return found
}

func (ks *DeployKeyStorage) Key(k string) (DeployKey, bool) {
	ks.RLock()
	defer ks.RUnlock()

	i, found := ks.keys[k]

	return i, found
}

func (ks *DeployKeyStorage) New() (DeployKey, error) {
	ks.Lock()
	defer ks.Unlock()

	nk := NewDeployKey()

	if ks.db != nil {
		m := map[string]DeployKey{}
		for i := range ks.keys {
			m[i] = ks.keys[i]
		}
		m[nk.Key()] = nk

		if err := saveDeployKeys(ks.db, m); err != nil {
			return DeployKey{}, err
		}
	}

	ks.keys[nk.Key()] = nk

	return nk, nil
}

func (ks *DeployKeyStorage) Revoke(k string) error {
	ks.Lock()
	defer ks.Unlock()

	if _, found := ks.keys[k]; !found {
		return util.NotFoundError
	}

	if ks.db != nil {
		m := map[string]DeployKey{}

		for i := range ks.keys {
			if i == k {
				continue
			}

			m[i] = ks.keys[i]
		}

		if err := saveDeployKeys(ks.db, m); err != nil {
			return err
		}
	}

	delete(ks.keys, k)

	return nil
}

func (ks *DeployKeyStorage) Len() int {
	return len(ks.keys)
}

func (ks *DeployKeyStorage) Traverse(callback func(DeployKey) bool) {
	ks.RLock()
	defer ks.RUnlock()

	for i := range ks.keys {
		if !callback(ks.keys[i]) {
			break
		}
	}
}

func loadDeployKeys(db storage.Database) (map[string]DeployKey, error) {
	var b []byte
	switch i, found, err := db.Info(DatabaseInfoDeployKeysKey); {
	case err != nil:
		return nil, errors.Wrap(err, "failed to load deploy keys from database")
	case !found:
		return nil, nil
	default:
		b = i
	}

	var uks map[string]DeployKey
	if err := db.Encoder().Unmarshal(b, &uks); err != nil {
		return nil, errors.Wrap(err, "failed to deocde deploy keys")
	}
	return uks, nil
}

func saveDeployKeys(db storage.Database, keys map[string]DeployKey) error {
	if i, err := db.Encoder().Marshal(keys); err != nil {
		return errors.Wrap(err, "failed to marshal deploy keys")
	} else if err := db.SetInfo(DatabaseInfoDeployKeysKey, i); err != nil {
		return errors.Wrap(err, "failed to save deploy keys")
	}

	return nil
}
