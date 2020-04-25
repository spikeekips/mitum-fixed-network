// +build mongodb

package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestProposalMakerMongodb(t *testing.T) {
	handler := new(testProposalMaker)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}
