// +build mongodb

package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestBlockV0DummyGeneratorMongodb(t *testing.T) {
	handler := new(testBlockV0DummyGenerator)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}
