// +build mongodb

package basicstates

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestStateSyncingMongodb(t *testing.T) {
	handler := new(testStateSyncing)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}
