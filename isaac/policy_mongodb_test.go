// +build mongodb

package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestPolicyMongodb(t *testing.T) {
	handler := new(testPolicy)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}
