package block

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	BaseBlockDataMapType   = hint.Type("base-blockdatamap")
	BaseBlockDataMapHint   = hint.NewHint(BaseBlockDataMapType, "v0.0.1")
	BaseBlockDataMapHinter = BaseBlockDataMap{BaseHinter: hint.NewBaseHinter(BaseBlockDataMapHint)}
)

var (
	BlockDataManifest        = "manifest"
	BlockDataOperations      = "operations"
	BlockDataOperationsTree  = "operations_tree"
	BlockDataStates          = "states"
	BlockDataStatesTree      = "states_tree"
	BlockDataINITVoteproof   = "init_voteproof"
	BlockDataACCEPTVoteproof = "accept_voteproof"
	BlockDataSuffrageInfo    = "suffrage_info"
	BlockDataProposal        = "proposal"
)

var BlockData = []string{
	BlockDataManifest,
	BlockDataOperations,
	BlockDataOperationsTree,
	BlockDataStates,
	BlockDataStatesTree,
	BlockDataINITVoteproof,
	BlockDataACCEPTVoteproof,
	BlockDataSuffrageInfo,
	BlockDataProposal,
}

type BaseBlockDataMap struct {
	hint.BaseHinter
	h          valuehash.Hash
	height     base.Height
	block      valuehash.Hash
	createdAt  time.Time
	items      map[string]BaseBlockDataMapItem
	writerHint hint.Hint
}

func NewBaseBlockDataMap(writerHint hint.Hint, height base.Height) BaseBlockDataMap {
	return BaseBlockDataMap{
		BaseHinter: hint.NewBaseHinter(BaseBlockDataMapHint),
		height:     height,
		createdAt:  localtime.UTCNow(),
		items: map[string]BaseBlockDataMapItem{
			BlockDataManifest:        {},
			BlockDataOperations:      {},
			BlockDataOperationsTree:  {},
			BlockDataStates:          {},
			BlockDataStatesTree:      {},
			BlockDataINITVoteproof:   {},
			BlockDataACCEPTVoteproof: {},
			BlockDataSuffrageInfo:    {},
			BlockDataProposal:        {},
		},
		writerHint: writerHint,
	}
}

func (bd BaseBlockDataMap) IsReadyToHash() error {
	if err := isvalid.Check([]isvalid.IsValider{
		bd.BaseHinter,
		bd.height,
		bd.block,
	}, nil, false); err != nil {
		return err
	}

	for dataType := range bd.items {
		if err := bd.items[dataType].IsValid(nil); err != nil {
			return errors.Wrapf(err, "invalid data type, %q found", dataType)
		}
	}

	return nil
}

func (bd BaseBlockDataMap) IsValid([]byte) error {
	if err := isvalid.Check([]isvalid.IsValider{bd.BaseHinter, bd.h}, nil, false); err != nil {
		return err
	}

	var isLocal *bool
	for i := range bd.items {
		i := IsLocalBlockDateItem(bd.items[i].URL())
		if isLocal == nil {
			isLocal = &i

			continue
		}

		if *isLocal != i {
			return errors.Errorf("all the items should be local or non-local")
		}
	}

	if err := bd.IsReadyToHash(); err != nil {
		return err
	}

	if !bd.h.Equal(bd.GenerateHash()) {
		return isvalid.InvalidError.Errorf("incorrect block data map hash")
	}

	return nil
}

func (bd BaseBlockDataMap) Hash() valuehash.Hash {
	return bd.h
}

func (bd BaseBlockDataMap) UpdateHash() (BaseBlockDataMap, error) {
	if err := bd.IsReadyToHash(); err != nil {
		return BaseBlockDataMap{}, err
	}

	bd.h = bd.GenerateHash()

	return bd, nil
}

func (bd BaseBlockDataMap) GenerateHash() valuehash.Hash {
	bs := make([][]byte, len(BlockData)+2)

	bs[0] = bd.height.Bytes()
	bs[1] = localtime.NewTime(bd.createdAt).Bytes()

	for i, dataType := range []string{
		BlockDataManifest,
		BlockDataOperations,
		BlockDataOperationsTree,
		BlockDataStates,
		BlockDataStatesTree,
		BlockDataINITVoteproof,
		BlockDataACCEPTVoteproof,
		BlockDataSuffrageInfo,
		BlockDataProposal,
	} {
		bs[2+i] = bd.items[dataType].Bytes()
	}

	return valuehash.NewSHA256(util.ConcatBytesSlice(bs...))
}

func (bd BaseBlockDataMap) Height() base.Height {
	return bd.height
}

func (bd BaseBlockDataMap) Writer() hint.Hint {
	return bd.writerHint
}

func (bd BaseBlockDataMap) CreatedAt() time.Time {
	return bd.createdAt
}

func (bd BaseBlockDataMap) IsLocal() bool {
	for i := range BlockData {
		if IsLocalBlockDateItem(bd.items[BlockData[i]].URL()) {
			return true
		}
	}

	return false
}

func (bd BaseBlockDataMap) Exists(b string) error {
	for i := range BlockData {
		item := bd.items[BlockData[i]]
		if err := item.Exists(b); err != nil {
			return err
		}
	}

	return nil
}

func (bd BaseBlockDataMap) Items() map[string]BaseBlockDataMapItem {
	return bd.items
}

func (bd BaseBlockDataMap) Item(dataType string) (BaseBlockDataMapItem, bool) {
	i, found := bd.items[dataType]

	return i, found
}

func (bd BaseBlockDataMap) SetItem(item BaseBlockDataMapItem) (BaseBlockDataMap, error) {
	if _, found := bd.items[item.Type()]; !found {
		return BaseBlockDataMap{}, errors.Errorf("unknown data type, %q of block data item", item.Type())
	}

	bd.items[item.Type()] = item

	return bd, nil
}

func (bd BaseBlockDataMap) Block() valuehash.Hash {
	return bd.block
}

func (bd BaseBlockDataMap) SetBlock(blk valuehash.Hash) BaseBlockDataMap {
	bd.block = blk

	return bd
}

func (bd BaseBlockDataMap) Manifest() BlockDataMapItem {
	return bd.items[BlockDataManifest]
}

func (bd BaseBlockDataMap) SetManifest(item BaseBlockDataMapItem) BaseBlockDataMap {
	bd.items[BlockDataManifest] = item

	return bd
}

func (bd BaseBlockDataMap) Operations() BlockDataMapItem {
	return bd.items[BlockDataOperations]
}

func (bd BaseBlockDataMap) SetOperations(item BaseBlockDataMapItem) BaseBlockDataMap {
	bd.items[BlockDataOperations] = item

	return bd
}

func (bd BaseBlockDataMap) OperationsTree() BlockDataMapItem {
	return bd.items[BlockDataOperationsTree]
}

func (bd BaseBlockDataMap) SetOperationsTree(item BaseBlockDataMapItem) BaseBlockDataMap {
	bd.items[BlockDataOperationsTree] = item

	return bd
}

func (bd BaseBlockDataMap) States() BlockDataMapItem {
	return bd.items[BlockDataStates]
}

func (bd BaseBlockDataMap) SetStates(item BaseBlockDataMapItem) BaseBlockDataMap {
	bd.items[BlockDataStates] = item

	return bd
}

func (bd BaseBlockDataMap) StatesTree() BlockDataMapItem {
	return bd.items[BlockDataStatesTree]
}

func (bd BaseBlockDataMap) SetStatesTree(item BaseBlockDataMapItem) BaseBlockDataMap {
	bd.items[BlockDataStatesTree] = item

	return bd
}

func (bd BaseBlockDataMap) INITVoteproof() BlockDataMapItem {
	return bd.items[BlockDataINITVoteproof]
}

func (bd BaseBlockDataMap) SetINITVoteproof(item BaseBlockDataMapItem) BaseBlockDataMap {
	bd.items[BlockDataINITVoteproof] = item

	return bd
}

func (bd BaseBlockDataMap) ACCEPTVoteproof() BlockDataMapItem {
	return bd.items[BlockDataACCEPTVoteproof]
}

func (bd BaseBlockDataMap) SetACCEPTVoteproof(item BaseBlockDataMapItem) BaseBlockDataMap {
	bd.items[BlockDataACCEPTVoteproof] = item

	return bd
}

func (bd BaseBlockDataMap) SuffrageInfo() BlockDataMapItem {
	return bd.items[BlockDataSuffrageInfo]
}

func (bd BaseBlockDataMap) SetSuffrageInfo(item BaseBlockDataMapItem) BaseBlockDataMap {
	bd.items[BlockDataSuffrageInfo] = item

	return bd
}

func (bd BaseBlockDataMap) Proposal() BlockDataMapItem {
	return bd.items[BlockDataProposal]
}

func (bd BaseBlockDataMap) SetProposal(item BaseBlockDataMapItem) BaseBlockDataMap {
	bd.items[BlockDataProposal] = item

	return bd
}

type BaseBlockDataMapItem struct {
	t        string
	checksum string
	url      string
}

func NewBaseBlockDataMapItem(dataType string, checksum string, url string) BaseBlockDataMapItem {
	return BaseBlockDataMapItem{
		t:        dataType,
		checksum: checksum,
		url:      url,
	}
}

func (bd BaseBlockDataMapItem) IsValid([]byte) error {
	if len(bd.t) < 1 {
		return isvalid.InvalidError.Errorf("empty data type of map item")
	}

	if len(bd.checksum) < 1 {
		return isvalid.InvalidError.Errorf("empty checksum of map item")
	}

	if n := strings.SplitN(bd.url, "://", 2); len(n) != 2 {
		return errors.Errorf("invalid url")
	} else if len(bd.URLBody()) < 1 {
		return isvalid.InvalidError.Errorf("empty url of map item")
	}

	return nil
}

func (bd BaseBlockDataMapItem) Bytes() []byte {
	return util.ConcatBytesSlice(
		[]byte(bd.t),
		[]byte(bd.checksum),
		[]byte(bd.url),
	)
}

func (bd BaseBlockDataMapItem) Type() string {
	return bd.t
}

func (bd BaseBlockDataMapItem) Checksum() string {
	return bd.checksum
}

func (bd BaseBlockDataMapItem) URL() string {
	return bd.url
}

func (bd BaseBlockDataMapItem) SetFile(p string) BaseBlockDataMapItem {
	bd.url = "file://" + p

	return bd
}

func (bd BaseBlockDataMapItem) SetURL(u string) BaseBlockDataMapItem {
	bd.url = u

	return bd
}

func (bd BaseBlockDataMapItem) URLBody() string {
	n := strings.SplitN(bd.url, "://", 2)
	switch {
	case len(n) != 2:
		return ""
	case len(bd.url) < len(n)+4:
		return ""
	default:
		return bd.url[len(n)+4:]
	}
}

func (bd BaseBlockDataMapItem) Exists(b string) error {
	if len(bd.URLBody()) < 1 {
		return os.ErrNotExist
	}

	if fi, err := os.Stat(filepath.Join(b, bd.URLBody())); err != nil {
		return err
	} else if fi.IsDir() {
		return os.ErrInvalid
	}

	return nil
}

func IsLocalBlockDateItem(u string) bool {
	switch {
	case strings.HasPrefix(u, "file://"):
		return true
	default:
		return false
	}
}
