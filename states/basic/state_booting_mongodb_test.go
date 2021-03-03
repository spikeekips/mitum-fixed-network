// +build mongodb

package basicstates

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestStateBootingMongodb(t *testing.T) {
	handler := new(testStateBooting)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}
