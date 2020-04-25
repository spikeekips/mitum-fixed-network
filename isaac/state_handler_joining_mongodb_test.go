// +build mongodb

package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestStateJoiningHandlerMongodb(t *testing.T) {
	handler := new(testStateJoiningHandler)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}
