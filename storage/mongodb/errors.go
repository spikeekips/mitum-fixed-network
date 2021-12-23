package mongodbstorage

import (
	"github.com/spikeekips/mitum/storage"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

func MergeError(err error) error {
	if err == nil {
		return nil
	}

	switch err.(type) {
	case topology.ConnectionError,
		topology.ServerSelectionError,
		topology.WaitQueueTimeoutError:
		err = storage.ConnectionError.Wrap(err)
	}

	return storage.MergeStorageError(err)
}
