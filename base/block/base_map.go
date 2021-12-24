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
	BaseBlockdataMapType   = hint.Type("base-blockdatamap")
	BaseBlockdataMapHint   = hint.NewHint(BaseBlockdataMapType, "v0.0.1")
	BaseBlockdataMapHinter = BaseBlockdataMap{BaseHinter: hint.NewBaseHinter(BaseBlockdataMapHint)}
)

var (
	BlockdataManifest        = "manifest"
	BlockdataOperations      = "operations"
	BlockdataOperationsTree  = "operations_tree"
	BlockdataStates          = "states"
	BlockdataStatesTree      = "states_tree"
	BlockdataINITVoteproof   = "init_voteproof"
	BlockdataACCEPTVoteproof = "accept_voteproof"
	BlockdataSuffrageInfo    = "suffrage_info"
	BlockdataProposal        = "proposal"
)

var Blockdata = []string{
	BlockdataManifest,
	BlockdataOperations,
	BlockdataOperationsTree,
	BlockdataStates,
	BlockdataStatesTree,
	BlockdataINITVoteproof,
	BlockdataACCEPTVoteproof,
	BlockdataSuffrageInfo,
	BlockdataProposal,
}

type BaseBlockdataMap struct {
	hint.BaseHinter
	h          valuehash.Hash
	height     base.Height
	block      valuehash.Hash
	createdAt  time.Time
	items      map[string]BaseBlockdataMapItem
	writerHint hint.Hint
}

func NewBaseBlockdataMap(writerHint hint.Hint, height base.Height) BaseBlockdataMap {
	return BaseBlockdataMap{
		BaseHinter: hint.NewBaseHinter(BaseBlockdataMapHint),
		height:     height,
		createdAt:  localtime.UTCNow(),
		items: map[string]BaseBlockdataMapItem{
			BlockdataManifest:        {},
			BlockdataOperations:      {},
			BlockdataOperationsTree:  {},
			BlockdataStates:          {},
			BlockdataStatesTree:      {},
			BlockdataINITVoteproof:   {},
			BlockdataACCEPTVoteproof: {},
			BlockdataSuffrageInfo:    {},
			BlockdataProposal:        {},
		},
		writerHint: writerHint,
	}
}

func (bd BaseBlockdataMap) IsReadyToHash() error {
	if err := isvalid.Check(nil, false, bd.BaseHinter, bd.height, bd.block); err != nil {
		return err
	}

	for dataType := range bd.items {
		if err := bd.items[dataType].IsValid(nil); err != nil {
			return errors.Wrapf(err, "invalid data type, %q found", dataType)
		}
	}

	return nil
}

func (bd BaseBlockdataMap) IsValid([]byte) error {
	if err := isvalid.Check(nil, false, bd.BaseHinter, bd.h); err != nil {
		return err
	}

	var isLocal *bool
	for i := range bd.items {
		i := IsLocalBlockdataItem(bd.items[i].URL())
		if isLocal == nil {
			isLocal = &i

			continue
		}

		if *isLocal != i {
			return isvalid.InvalidError.Errorf("all the items should be local or non-local")
		}
	}

	if err := bd.IsReadyToHash(); err != nil {
		return isvalid.InvalidError.Wrap(err)
	}

	if !bd.h.Equal(bd.GenerateHash()) {
		return isvalid.InvalidError.Errorf("incorrect block data map hash")
	}

	return nil
}

func (bd BaseBlockdataMap) Hash() valuehash.Hash {
	return bd.h
}

func (bd BaseBlockdataMap) UpdateHash() (BaseBlockdataMap, error) {
	if err := bd.IsReadyToHash(); err != nil {
		return BaseBlockdataMap{}, err
	}

	bd.h = bd.GenerateHash()

	return bd, nil
}

func (bd BaseBlockdataMap) GenerateHash() valuehash.Hash {
	bs := make([][]byte, len(Blockdata)+2)

	bs[0] = bd.height.Bytes()
	bs[1] = localtime.NewTime(bd.createdAt).Bytes()

	for i, dataType := range []string{
		BlockdataManifest,
		BlockdataOperations,
		BlockdataOperationsTree,
		BlockdataStates,
		BlockdataStatesTree,
		BlockdataINITVoteproof,
		BlockdataACCEPTVoteproof,
		BlockdataSuffrageInfo,
		BlockdataProposal,
	} {
		bs[2+i] = bd.items[dataType].Bytes()
	}

	return valuehash.NewSHA256(util.ConcatBytesSlice(bs...))
}

func (bd BaseBlockdataMap) Height() base.Height {
	return bd.height
}

func (bd BaseBlockdataMap) Writer() hint.Hint {
	return bd.writerHint
}

func (bd BaseBlockdataMap) CreatedAt() time.Time {
	return bd.createdAt
}

func (bd BaseBlockdataMap) IsLocal() bool {
	for i := range Blockdata {
		if IsLocalBlockdataItem(bd.items[Blockdata[i]].URL()) {
			return true
		}
	}

	return false
}

func (bd BaseBlockdataMap) Exists(b string) error {
	for i := range Blockdata {
		item := bd.items[Blockdata[i]]
		if err := item.Exists(b); err != nil {
			return err
		}
	}

	return nil
}

func (bd BaseBlockdataMap) Items() map[string]BaseBlockdataMapItem {
	return bd.items
}

func (bd BaseBlockdataMap) Item(dataType string) (BaseBlockdataMapItem, bool) {
	i, found := bd.items[dataType]

	return i, found
}

func (bd BaseBlockdataMap) SetItem(item BaseBlockdataMapItem) (BaseBlockdataMap, error) {
	if _, found := bd.items[item.Type()]; !found {
		return BaseBlockdataMap{}, errors.Errorf("unknown data type, %q of block data item", item.Type())
	}

	bd.items[item.Type()] = item

	return bd, nil
}

func (bd BaseBlockdataMap) Block() valuehash.Hash {
	return bd.block
}

func (bd BaseBlockdataMap) SetBlock(blk valuehash.Hash) BaseBlockdataMap {
	bd.block = blk

	return bd
}

func (bd BaseBlockdataMap) Manifest() BlockdataMapItem {
	return bd.items[BlockdataManifest]
}

func (bd BaseBlockdataMap) SetManifest(item BaseBlockdataMapItem) BaseBlockdataMap {
	bd.items[BlockdataManifest] = item

	return bd
}

func (bd BaseBlockdataMap) Operations() BlockdataMapItem {
	return bd.items[BlockdataOperations]
}

func (bd BaseBlockdataMap) SetOperations(item BaseBlockdataMapItem) BaseBlockdataMap {
	bd.items[BlockdataOperations] = item

	return bd
}

func (bd BaseBlockdataMap) OperationsTree() BlockdataMapItem {
	return bd.items[BlockdataOperationsTree]
}

func (bd BaseBlockdataMap) SetOperationsTree(item BaseBlockdataMapItem) BaseBlockdataMap {
	bd.items[BlockdataOperationsTree] = item

	return bd
}

func (bd BaseBlockdataMap) States() BlockdataMapItem {
	return bd.items[BlockdataStates]
}

func (bd BaseBlockdataMap) SetStates(item BaseBlockdataMapItem) BaseBlockdataMap {
	bd.items[BlockdataStates] = item

	return bd
}

func (bd BaseBlockdataMap) StatesTree() BlockdataMapItem {
	return bd.items[BlockdataStatesTree]
}

func (bd BaseBlockdataMap) SetStatesTree(item BaseBlockdataMapItem) BaseBlockdataMap {
	bd.items[BlockdataStatesTree] = item

	return bd
}

func (bd BaseBlockdataMap) INITVoteproof() BlockdataMapItem {
	return bd.items[BlockdataINITVoteproof]
}

func (bd BaseBlockdataMap) SetINITVoteproof(item BaseBlockdataMapItem) BaseBlockdataMap {
	bd.items[BlockdataINITVoteproof] = item

	return bd
}

func (bd BaseBlockdataMap) ACCEPTVoteproof() BlockdataMapItem {
	return bd.items[BlockdataACCEPTVoteproof]
}

func (bd BaseBlockdataMap) SetACCEPTVoteproof(item BaseBlockdataMapItem) BaseBlockdataMap {
	bd.items[BlockdataACCEPTVoteproof] = item

	return bd
}

func (bd BaseBlockdataMap) SuffrageInfo() BlockdataMapItem {
	return bd.items[BlockdataSuffrageInfo]
}

func (bd BaseBlockdataMap) SetSuffrageInfo(item BaseBlockdataMapItem) BaseBlockdataMap {
	bd.items[BlockdataSuffrageInfo] = item

	return bd
}

func (bd BaseBlockdataMap) Proposal() BlockdataMapItem {
	return bd.items[BlockdataProposal]
}

func (bd BaseBlockdataMap) SetProposal(item BaseBlockdataMapItem) BaseBlockdataMap {
	bd.items[BlockdataProposal] = item

	return bd
}

type BaseBlockdataMapItem struct {
	t        string
	checksum string
	url      string
}

func NewBaseBlockdataMapItem(dataType string, checksum string, url string) BaseBlockdataMapItem {
	return BaseBlockdataMapItem{
		t:        dataType,
		checksum: checksum,
		url:      url,
	}
}

func (bd BaseBlockdataMapItem) IsValid([]byte) error {
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

func (bd BaseBlockdataMapItem) Bytes() []byte {
	return util.ConcatBytesSlice(
		[]byte(bd.t),
		[]byte(bd.checksum),
		[]byte(bd.url),
	)
}

func (bd BaseBlockdataMapItem) Type() string {
	return bd.t
}

func (bd BaseBlockdataMapItem) Checksum() string {
	return bd.checksum
}

func (bd BaseBlockdataMapItem) URL() string {
	return bd.url
}

func (bd BaseBlockdataMapItem) SetFile(p string) BaseBlockdataMapItem {
	bd.url = "file://" + p

	return bd
}

func (bd BaseBlockdataMapItem) SetURL(u string) BaseBlockdataMapItem {
	bd.url = u

	return bd
}

func (bd BaseBlockdataMapItem) URLBody() string {
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

func (bd BaseBlockdataMapItem) Exists(b string) error {
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

func IsLocalBlockdataItem(u string) bool {
	switch {
	case strings.HasPrefix(u, "file://"):
		return true
	default:
		return false
	}
}
