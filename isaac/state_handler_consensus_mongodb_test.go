// +build mongodb

package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestStateConsensusHandlerMongodb(t *testing.T) {
	handler := new(testStateConsensusHandler)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}
