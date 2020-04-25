// +build mongodb

package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestGenesisBlockV0Mongodb(t *testing.T) {
	handler := new(testGenesisBlockV0)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}
