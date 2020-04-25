// +build mongodb

package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestGeneralSyncerMongodb(t *testing.T) {
	handler := new(testGeneralSyncer)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}
