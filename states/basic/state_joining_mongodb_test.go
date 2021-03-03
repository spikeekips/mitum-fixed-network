// +build mongodb

package basicstates

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestStateJoiningMongodb(t *testing.T) {
	handler := new(testStateJoining)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}
