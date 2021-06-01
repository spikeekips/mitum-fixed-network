// +build mongodb

package deploy

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestDeployKeyHandlersMongodb(t *testing.T) {
	handler := new(testDeployKeyHandlers)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}
