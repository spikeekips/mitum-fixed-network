package contest_module

import (
	"crypto/rand"
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
)

type testFixedProposerSuffrage struct {
	suite.Suite
}

func (t *testFixedProposerSuffrage) TestSelectNodes() {
	n := 6
	var a uint = 4

	var nodes []node.Node
	for i := 0; i < n; i++ {
		nodes = append(nodes, node.NewRandomHome())
	}
	fs := NewFixedProposerSuffrage(nodes[0], a, nodes...)

	for i := 0; i < 50; i++ {
		h, _ := rand.Int(rand.Reader, big.NewInt(1000))
		r, _ := rand.Int(rand.Reader, big.NewInt(1000))
		acting := fs.Acting(isaac.NewBlockHeight(uint64(h.Int64())), isaac.Round(r.Int64()))
		t.Equal(int(a), len(acting.Nodes()))

		// check node duplication
		var k []node.Node
		for _, i := range acting.Nodes() {
			for _, e := range k {
				if i.Equal(e) {
					t.Error(errors.New("duplication found"))
					return
				}
			}
			k = append(k, i)
		}
	}
}

func TestFixedProposerSuffrage(t *testing.T) {
	suite.Run(t, new(testFixedProposerSuffrage))
}
