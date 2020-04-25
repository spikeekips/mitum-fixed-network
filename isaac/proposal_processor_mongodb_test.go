// +build mongodb

package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestProposalProcessorMongodb(t *testing.T) {
	handler := new(testProposalProcessor)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}
