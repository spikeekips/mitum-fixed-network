// +build mongodb

package deploy

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestDeployKeyStorageWithMongodb(t *testing.T) {
	handler := new(testDeployKeyStorageWithDatabase)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}
