package localfs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

func LoadBlock(st *Blockdata, height base.Height) (block.BaseBlockdataMap, block.Block, error) { // nolint
	var bdm block.BaseBlockdataMap
	if found, err := st.Exists(height); err != nil {
		return bdm, nil, err
	} else if !found {
		return bdm, nil, util.NotFoundError.Errorf("block data %d not found", height)
	}

	prepath := st.heightDirectory(height, true)

	return LoadBlockByPath(st, prepath)
}

func LoadBlockByPath(st *Blockdata, prepath string) (block.BaseBlockdataMap, block.Block, error) { // nolint
	blk := (interface{})(block.EmptyBlockV0()).(block.BlockUpdater)

	var bdm block.BaseBlockdataMap
	var mapItems []block.BaseBlockdataMapItem
	if m, r, err := LoadData(prepath, block.BlockdataManifest); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadManifest(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetManifest(i)
		mapItems = append(mapItems, m)
	}

	if m, r, err := LoadData(prepath, block.BlockdataOperationsTree); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadOperationsTree(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetOperationsTree(i)
		mapItems = append(mapItems, m)
	}

	if m, r, err := LoadData(prepath, block.BlockdataOperations); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadOperations(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetOperations(i)
		mapItems = append(mapItems, m)
	}

	if m, r, err := LoadData(prepath, block.BlockdataStatesTree); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadStatesTree(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetStatesTree(i)
		mapItems = append(mapItems, m)
	}

	if m, r, err := LoadData(prepath, block.BlockdataStates); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadStates(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetStates(i)
		mapItems = append(mapItems, m)
	}

	if m, r, err := LoadData(prepath, block.BlockdataINITVoteproof); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadINITVoteproof(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetINITVoteproof(i)
		mapItems = append(mapItems, m)
	}

	if m, r, err := LoadData(prepath, block.BlockdataACCEPTVoteproof); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadACCEPTVoteproof(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetACCEPTVoteproof(i)
		mapItems = append(mapItems, m)
	}

	if m, r, err := LoadData(prepath, block.BlockdataSuffrageInfo); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadSuffrageInfo(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetSuffrageInfo(i)
		mapItems = append(mapItems, m)
	}

	if m, r, err := LoadData(prepath, block.BlockdataProposal); err != nil {
		return bdm, nil, err
	} else if i, err := st.Writer().ReadProposal(r); err != nil {
		return bdm, nil, err
	} else {
		blk = blk.SetProposal(i)
		mapItems = append(mapItems, m)
	}

	bdm = block.NewBaseBlockdataMap(st.Writer().Hint(), blk.Height())
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

func OpenFile(prepath, dataType string) (string, io.ReadCloser, error) {
	g := filepath.Join(prepath, fmt.Sprintf(BlockFileGlobFormats, dataType))

	var f string
	switch matches, err := filepath.Glob(g); {
	case err != nil:
		return "", nil, storage.MergeStorageError(err)
	case len(matches) < 1:
		return "", nil, util.NotFoundError.Errorf("block data, %q not found", g)
	default:
		f = matches[0]
	}

	i, err := os.Open(filepath.Clean(f))
	if err != nil {
		return "", nil, storage.MergeStorageError(err)
	}

	return f, i, nil
}

func LoadData(prepath, dataType string) (block.BaseBlockdataMapItem, io.ReadCloser, error) {
	var m block.BaseBlockdataMapItem

	f, i, err := OpenFile(prepath, dataType)
	if err != nil {
		return m, nil, err
	}

	j, err := util.NewGzipReader(i)
	if err != nil {
		return m, nil, storage.MergeStorageError(err)
	}

	k, err := NewBaseBlockdataMapItem(f)
	if err != nil {
		return m, nil, err
	}

	return k, j, nil
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

	return "/" + strings.Join(sl, "/")
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
		return base.NilHeight, "", "", errors.Errorf("invalid file format: %s", s)
	}

	if strings.HasPrefix(o, "-") {
		a *= -1
	}

	return base.Height(a), b, c, nil
}

func NewBaseBlockdataMapItem(f string) (block.BaseBlockdataMapItem, error) {
	height, dataType, checksum, err := ParseDataFileName(f)
	if err != nil {
		return block.BaseBlockdataMapItem{}, err
	}

	return block.NewBaseBlockdataMapItem(
		dataType,
		checksum,
		"file://"+filepath.Join(HeightDirectory(height), filepath.Base(f)),
	), nil
}
