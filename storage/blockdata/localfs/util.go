package localfs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
)

func LoadBlock(st *BlockData, height base.Height) (block.BaseBlockDataMap, block.Block, error) { // nolint
	var bdm block.BaseBlockDataMap
	blk := (interface{})(block.BlockV0{}).(block.BlockUpdater)

	var mapItems []block.BaseBlockDataMapItem
	if m, r, err := LoadData(st, height, block.BlockDataManifest); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadManifest(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetManifest(i)
		mapItems = append(mapItems, m)
	}

	if m, r, err := LoadData(st, height, block.BlockDataOperationsTree); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadOperationsTree(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetOperationsTree(i)
		mapItems = append(mapItems, m)
	}

	if m, r, err := LoadData(st, height, block.BlockDataOperations); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadOperations(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetOperations(i)
		mapItems = append(mapItems, m)
	}

	if m, r, err := LoadData(st, height, block.BlockDataStatesTree); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadStatesTree(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetStatesTree(i)
		mapItems = append(mapItems, m)
	}

	if m, r, err := LoadData(st, height, block.BlockDataStates); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadStates(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetStates(i)
		mapItems = append(mapItems, m)
	}

	if m, r, err := LoadData(st, height, block.BlockDataINITVoteproof); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadINITVoteproof(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetINITVoteproof(i)
		mapItems = append(mapItems, m)
	}

	if m, r, err := LoadData(st, height, block.BlockDataACCEPTVoteproof); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadACCEPTVoteproof(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetACCEPTVoteproof(i)
		mapItems = append(mapItems, m)
	}

	if m, r, err := LoadData(st, height, block.BlockDataSuffrageInfo); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadSuffrageInfo(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetSuffrageInfo(i)
		mapItems = append(mapItems, m)
	}

	if m, r, err := LoadData(st, height, block.BlockDataProposal); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadProposal(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetProposal(i)
		mapItems = append(mapItems, m)
	}

	bdm = block.NewBaseBlockDataMap(st.Writer().Hint(), height)
	bdm = bdm.SetBlock(blk.Hash())
	for i := range mapItems {
		j, err := bdm.SetItem(mapItems[i])
		if err != nil {
			return bdm, nil, err
		}
		bdm = j
	}

	if i, err := bdm.UpdateHash(); err != nil {
		return bdm, nil, err
	} else if err := i.IsValid(nil); err != nil {
		return bdm, nil, err
	} else {
		return i, blk, nil
	}
}

func LoadData(st *BlockData, height base.Height, dataType string) (block.BaseBlockDataMapItem, io.ReadCloser, error) {
	var m block.BaseBlockDataMapItem
	if found, err := st.Exists(height); err != nil {
		return m, nil, err
	} else if !found {
		return m, nil, util.NotFoundError.Errorf("block data %d not found", height)
	}

	g := filepath.Join(st.heightDirectory(height, true), fmt.Sprintf(BlockFileGlobFormats, height, dataType))

	var f string
	switch matches, err := filepath.Glob(g); {
	case err != nil:
		return m, nil, storage.WrapStorageError(err)
	case len(matches) < 1:
		return m, nil, util.NotFoundError.Errorf("block data, %q(%d) not found", dataType, height)
	default:
		f = matches[0]
	}

	if i, err := os.Open(filepath.Clean(f)); err != nil {
		return m, nil, storage.WrapStorageError(err)
	} else if j, err := util.NewGzipReader(i); err != nil {
		return m, nil, storage.WrapStorageError(err)
	} else if k, err := NewBaseBlockDataMapItem(f); err != nil {
		return m, nil, err
	} else {
		return k, j, nil
	}
}

func HeightDirectory(height base.Height) string {
	h := height.String()
	if height < 0 {
		h = strings.ReplaceAll(h, "-", "_")
	}

	p := fmt.Sprintf(BlockDirectoryHeightFormat, h)

	sl := make([]string, 7)
	var i int
	for {
		e := (i * 3) + 3
		if e > len(p) {
			e = len(p)
		}

		s := p[i*3 : e]
		if len(s) < 1 {
			break
		}

		sl[i] = s

		if len(s) < 3 {
			break
		}

		i++
	}

	return "/" + filepath.Join(strings.Join(sl, "/"))
}

func ParseDataFileName(s string) (base.Height, string /* data type */, string /* checksum */, error) {
	o := filepath.Base(s)
	y := strings.ReplaceAll(o, "-", " ")
	y = strings.ReplaceAll(y, ".", " ")

	var a int64
	var b string
	var c string
	if n, err := fmt.Sscanf(y+"\n", "%d %s %s", &a, &b, &c); err != nil {
		return base.NilHeight, "", "", err
	} else if n != 3 {
		return base.NilHeight, "", "", xerrors.Errorf("invalid file format: %s", s)
	}

	if strings.HasPrefix(o, "-") {
		a *= -1
	}

	return base.Height(a), b, c, nil
}

func NewBaseBlockDataMapItem(f string) (block.BaseBlockDataMapItem, error) {
	height, dataType, checksum, err := ParseDataFileName(f)
	if err != nil {
		return block.BaseBlockDataMapItem{}, err
	}

	return block.NewBaseBlockDataMapItem(
		dataType,
		checksum,
		"file://"+filepath.Join(HeightDirectory(height), filepath.Base(f)),
	), nil
}
