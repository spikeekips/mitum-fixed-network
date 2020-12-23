// +build test

package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	channetwork "github.com/spikeekips/mitum/network/gochan"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

type baseTestSyncer struct {
	baseTestStateHandler
}

func (t *baseTestSyncer) generateBlocks(locals []*Local, targetHeight base.Height) {
	bg, err := NewDummyBlocksV0Generator(
		locals[0],
		targetHeight,
		t.suffrage(locals[0], locals...),
		locals,
	)
	t.NoError(err)
	t.NoError(bg.Generate(false))
}

func (t *baseTestSyncer) emptyLocal() *Local {
	lst := t.Storage(nil, nil)
	localNode := channetwork.RandomLocalNode(util.UUID().String())
	blockfs := t.BlockFS(t.JSONEnc)

	local, err := NewLocal(lst, blockfs, localNode, TestNetworkID)
	t.NoError(err)

	t.NoError(local.Initialize())

	return local
}

func (t *baseTestStateHandler) setup(local *Local, others []*Local) {
	var nodes []*Local = []*Local{local}
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
		nch := st.Node().Channel().(*channetwork.Channel)
		nch.SetGetManifestsHandler(func(heights []base.Height) ([]block.Manifest, error) {
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

		nch.SetGetBlocksHandler(func(heights []base.Height) ([]block.Block, error) {
			var bs []block.Block
			for _, h := range heights {
				if blk, err := st.BlockFS().Load(h); err != nil {
					if xerrors.Is(err, storage.NotFoundError) {
						break
					}

					return nil, err
				} else {
					bs = append(bs, blk)
				}
			}

			return bs, nil
		})
	}
}
