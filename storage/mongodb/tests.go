// +build test mongodb

package mongodbstorage

import (
	"fmt"
	"os"

	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
)

func (st *Database) SetLastBlock(m block.Block) {
	_ = st.setLastBlock(m, true, false)
}

var BaseTestMongodbURI = "mongodb://localhost:27017"

func TestMongodbURI() string {
	uri := "localhost"
	if s := os.Getenv("MITUM_TEST_MONGODB_URI"); len(s) > 0 {
		uri = s
	}

	return fmt.Sprintf("mongodb://%s/t_%s", uri, util.UUID().String())
}

func (st *Database) OperationFactCache() cache.Cache {
	return st.operationFactCache
}
