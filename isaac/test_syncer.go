// +build test

package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	channetwork "github.com/spikeekips/mitum/network/gochan"
	"github.com/spikeekips/mitum/util"
)

type baseTestSyncer struct {
	baseTestStateHandler
}

func (t *baseTestSyncer) generateBlocks(localstates []*Localstate, targetHeight base.Height) {
	bg, err := NewDummyBlocksV0Generator(
		localstates[0],
		targetHeight,
		t.suffrage(localstates[0], localstates...),
		localstates,
	)
	t.NoError(err)
	t.NoError(bg.Generate(false))
}

func (t *baseTestSyncer) emptyLocalstate() *Localstate {
	lst := t.Storage(nil, nil)
	localNode := RandomLocalNode(util.UUID().String(), nil)
	localstate, err := NewLocalstate(lst, localNode, TestNetworkID)
	t.NoError(err)

	return localstate
}

func (t *baseTestStateHandler) setup(local *Localstate, others []*Localstate) {
	var nodes []*Localstate = []*Localstate{local}
	nodes = append(nodes, others...)

	lastHeight := t.lastManifest(local.Storage()).Height()

	for _, l := range nodes {
		t.NoError(l.Storage().Clean())
	}

	bg, err := NewDummyBlocksV0Generator(
		local,
		lastHeight,
		t.suffrage(local, nodes...),
		nodes,
	)
	t.NoError(err)
	t.NoError(bg.Generate(true))

	for _, st := range nodes {
		nch := st.Node().Channel().(*channetwork.NetworkChanChannel)
		nch.SetGetManifests(func(heights []base.Height) ([]block.Manifest, error) {
			var bs []block.Manifest
			for _, h := range heights {
				m, found, err := st.Storage().ManifestByHeight(h)
				if !found {
					break
				} else if err != nil {
					return nil, err
				}

				bs = append(bs, m)
			}

			return bs, nil
		})

		nch.SetGetBlocks(func(heights []base.Height) ([]block.Block, error) {
			var bs []block.Block
			for _, h := range heights {
				m, found, err := st.Storage().BlockByHeight(h)
				if !found {
					break
				} else if err != nil {
					return nil, err
				}

				bs = append(bs, m)
			}

			return bs, nil
		})
	}
}
