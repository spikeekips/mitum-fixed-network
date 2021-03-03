// +build mongodb

package basicstates

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestStateConsensusMongodb(t *testing.T) {
	handler := new(testStateConsensus)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}
