// +build mongodb

package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestStateSyncingHandlerMongodb(t *testing.T) {
	handler := new(testStateSyncingHandler)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}
