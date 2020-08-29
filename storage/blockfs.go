package storage

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/tree"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/errors"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	heightFormat = "%021s"
	blockFiles   = []string{
		"manifest",
		"operations",
		"states",
		"init_voteproof",
		"accept_voteproof",
		"suffrage",
		"proposal",
	}
	regBlockFilename = regexp.MustCompile(`^(?i)(?P<height>[0-9_][0-9_]*)\-(?P<block_hash>[a-z0-9][a-z0-9]*)` +
		`\-(?P<name>[\w][\w]*)\-(?P<checksum>[a-z0-9][a-z0-9]*)\.([a-z0-9][a-z0-9]*)\.gz$`)
)

type BlockFS struct {
	sync.RWMutex
	*logging.Logging
	fileLock            map[string]*sync.Mutex
	fs                  FS
	enc                 encoder.Encoder
	lastINITVoteproof   base.Voteproof
	lastACCEPTVoteproof base.Voteproof
}

func NewBlockFS(fs FS, enc *jsonenc.Encoder) *BlockFS {
	fileLock := map[string]*sync.Mutex{}
	for _, s := range blockFiles {
		fileLock[s] = &sync.Mutex{}
	}

	return &BlockFS{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "block-fs-storage")
		}),
		fileLock: fileLock,
		fs:       fs,
		enc:      enc,
	}
}

func (bs *BlockFS) FS() FS {
	return bs.fs
}

func (bs *BlockFS) OpenManifest(height base.Height) (io.ReadCloser, bool, error) {
	return bs.open(height, "manifest")
}

func (bs *BlockFS) OpenOperations(height base.Height) (io.ReadCloser, bool, error) {
	return bs.open(height, "operations")
}

func (bs *BlockFS) OpenStates(height base.Height) (io.ReadCloser, bool, error) {
	return bs.open(height, "states")
}

func (bs *BlockFS) OpenINITVoteproof(height base.Height) (io.ReadCloser, bool, error) {
	return bs.open(height, "init_voteproof")
}

func (bs *BlockFS) OpenACCEPTVoteproof(height base.Height) (io.ReadCloser, bool, error) {
	return bs.open(height, "accept_voteproof")
}

func (bs *BlockFS) OpenSuffrage(height base.Height) (io.ReadCloser, bool, error) {
	return bs.open(height, "suffrage")
}

func (bs *BlockFS) OpenProposal(height base.Height) (io.ReadCloser, bool, error) {
	return bs.open(height, "proposal")
}

func (bs *BlockFS) Load(height base.Height) (block.Block, error) {
	var manifest block.Manifest
	if i, err := bs.LoadManifest(height); err != nil {
		return nil, err
	} else {
		manifest = i
	}

	var ops *tree.AVLTree
	if i, err := bs.LoadOperations(height); err != nil {
		return nil, err
	} else {
		ops = i
	}

	var states *tree.AVLTree
	if i, err := bs.LoadStates(height); err != nil {
		return nil, err
	} else {
		states = i
	}

	var ivp base.Voteproof
	if i, err := bs.LoadINITVoteproof(height); err != nil {
		return nil, err
	} else {
		ivp = i
	}

	var avp base.Voteproof
	if i, err := bs.LoadACCEPTVoteproof(height); err != nil {
		return nil, err
	} else {
		avp = i
	}

	var suffrage block.SuffrageInfo
	if i, err := bs.LoadSuffrage(height); err != nil {
		return nil, err
	} else {
		suffrage = i
	}

	var proposal ballot.Proposal
	if i, err := bs.LoadProposal(height); err != nil {
		return nil, err
	} else {
		proposal = i
	}

	blk := block.BlockV0{}

	return blk.SetManifest(manifest).
		SetOperations(ops).
		SetStates(states).
		SetINITVoteproof(ivp).
		SetACCEPTVoteproof(avp).
		SetSuffrageInfo(suffrage).
		SetProposal(proposal), nil
}

func (bs *BlockFS) LoadManifest(height base.Height) (block.Manifest, error) {
	if hinter, err := bs.load(height, "manifest"); err != nil {
		return nil, err
	} else if i, ok := hinter.(block.Manifest); !ok {
		return nil, xerrors.Errorf("not block.Manifest, %T", hinter)
	} else {
		return i, nil
	}
}

func (bs *BlockFS) LoadOperations(height base.Height) (*tree.AVLTree, error) {
	if hinter, err := bs.load(height, "operations"); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(tree.AVLTree); !ok {
		return nil, xerrors.Errorf("not operations, *tree.AVLTree, %T", hinter)
	} else {
		return &i, nil
	}
}

func (bs *BlockFS) LoadStates(height base.Height) (*tree.AVLTree, error) {
	if hinter, err := bs.load(height, "states"); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(tree.AVLTree); !ok {
		return nil, xerrors.Errorf("not states, *tree.AVLTree, %T", hinter)
	} else {
		return &i, nil
	}
}

func (bs *BlockFS) LoadINITVoteproof(height base.Height) (base.Voteproof, error) {
	if hinter, err := bs.load(height, "init_voteproof"); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(base.Voteproof); !ok {
		return nil, xerrors.Errorf("not init voteproof, %T", hinter)
	} else {
		return i, nil
	}
}

func (bs *BlockFS) LoadACCEPTVoteproof(height base.Height) (base.Voteproof, error) {
	if hinter, err := bs.load(height, "accept_voteproof"); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(base.Voteproof); !ok {
		return nil, xerrors.Errorf("not accept voteproof, %T", hinter)
	} else {
		return i, nil
	}
}

func (bs *BlockFS) LoadSuffrage(height base.Height) (block.SuffrageInfo, error) {
	if hinter, err := bs.load(height, "suffrage"); err != nil {
		return nil, err
	} else if i, ok := hinter.(block.SuffrageInfo); !ok {
		return nil, xerrors.Errorf("not block.SuffrageInfo, %T", hinter)
	} else {
		return i, nil
	}
}

func (bs *BlockFS) LoadProposal(height base.Height) (ballot.Proposal, error) {
	if hinter, err := bs.load(height, "proposal"); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(ballot.Proposal); !ok {
		return nil, xerrors.Errorf("not Proposal, %T", hinter)
	} else {
		return i, nil
	}
}

func (bs *BlockFS) Add(blk block.Block) error {
	height := blk.Height()
	bh := blk.Hash()

	unstaged := bs.unstaged(height, bh)
	if err := bs.fs.RemoveDirectory(unstaged); err != nil {
		if !xerrors.Is(err, NotFoundError) {
			return err
		}
	}

	if err := bs.fs.CreateDirectory(unstaged); err != nil {
		if !xerrors.Is(err, FoundError) {
			return err
		}
	}

	var wg sync.WaitGroup
	wg.Add(7)

	errchan := make(chan error, 7)

	f := func(name string, i interface{}) {
		defer wg.Done()

		errchan <- bs.add(height, bh, name, i)
	}

	go f("manifest", blk.Manifest())
	go f("operations", blk.Operations())
	go f("states", blk.States())
	go f("init_voteproof", blk.ConsensusInfo().INITVoteproof())
	go f("accept_voteproof", blk.ConsensusInfo().ACCEPTVoteproof())
	go f("suffrage", blk.ConsensusInfo().SuffrageInfo())
	go f("proposal", blk.ConsensusInfo().Proposal())

	wg.Wait()
	close(errchan)

	var err error
	for e := range errchan {
		if e != nil {
			continue
		}

		err = e

		break
	}

	if err == nil {
		return err
	}

	err0 := errors.NewError("failed to save block data").Wrap(err)
	if err1 := bs.Cancel(height, bh); err1 != nil {
		return err0.Wrap(err1)
	}

	return err0
}

func (bs *BlockFS) AddManifest(height base.Height, bh valuehash.Hash, i block.Manifest) error {
	return bs.add(height, bh, "manifest", i)
}

func (bs *BlockFS) AddOperations(height base.Height, bh valuehash.Hash, i *tree.AVLTree) error {
	return bs.add(height, bh, "operations", i)
}

func (bs *BlockFS) AddStates(height base.Height, bh valuehash.Hash, i *tree.AVLTree) error {
	return bs.add(height, bh, "states", i)
}

func (bs *BlockFS) AddINITVoteproof(height base.Height, bh valuehash.Hash, i base.Voteproof) error {
	return bs.add(height, bh, "init_voteproof", i)
}

func (bs *BlockFS) AddACCEPTVoteproof(height base.Height, bh valuehash.Hash, i base.Voteproof) error {
	return bs.add(height, bh, "accept_voteproof", i)
}

func (bs *BlockFS) AddSuffrage(height base.Height, bh valuehash.Hash, i block.SuffrageInfo) error {
	return bs.add(height, bh, "suffrage", i)
}

func (bs *BlockFS) AddProposal(height base.Height, bh valuehash.Hash, i ballot.Proposal) error {
	return bs.add(height, bh, "proposal", i)
}

func (bs *BlockFS) Commit(height base.Height, bh valuehash.Hash) error {
	bs.Lock()
	defer bs.Unlock()

	_ = bs.remove(height)

	unstaged := bs.unstaged(height, bh)
	if err := bs.existsWithHash(unstaged, height, bh); err != nil {
		return err
	}

	if err := bs.fs.Rename(unstaged, bs.base(height)); err != nil {
		return bs.Cancel(height, bh)
	} else if err := bs.setLast(height); err != nil {
		return err
	} else {
		return nil
	}
}

func (bs *BlockFS) AddAndCommit(blk block.Block) error {
	if err := bs.Add(blk); err != nil {
		return err
	} else if err := bs.Commit(blk.Height(), blk.Hash()); err != nil {
		return err
	} else {
		return nil
	}
}

func (bs *BlockFS) Cancel(height base.Height, h valuehash.Hash) error {
	bs.Lock()
	defer bs.Unlock()

	unstaged := bs.unstaged(height, h)
	if err := bs.existsWithHash(unstaged, height, h); err != nil {
		if !xerrors.Is(err, NotFoundError) {
			return err
		}
	} else if err := bs.fs.RemoveDirectory(unstaged); err != nil {
		return err
	}

	return nil
}

func (bs *BlockFS) Remove(height base.Height) error {
	bs.Lock()
	defer bs.Unlock()

	return bs.remove(height)
}

func (bs *BlockFS) Clean(remove bool) error {
	bs.Lock()
	defer bs.Unlock()

	return bs.fs.Clean(remove)
}

func (bs *BlockFS) CleanByHeight(height base.Height) error {
	bs.Lock()
	defer bs.Unlock()

	if err := bs.cleanByHeight(height); err != nil {
		return err
	}

	return bs.setLast(height - 1)
}

func (bs *BlockFS) Exists(height base.Height) (valuehash.Hash, error) {
	var h valuehash.Hash
	founds := map[string]bool{}
	for _, f := range blockFiles {
		founds[f] = false
	}

	if err := bs.walk(bs.base(height), height, nil, func(fp string, fi os.FileInfo) error {
		if _, ph, name, _, _, err := bs.parseFilename(fi.Name()); err != nil {
			return nil
		} else if _, found := founds[name]; !found {
			return nil
		} else {
			h = ph
			founds[name] = true
		}

		return nil
	}); err != nil {
		return nil, err
	}

	for _, f := range founds {
		if !f {
			return nil, NotFoundError.Errorf("no block files found")
		}
	}

	return h, nil
}

func (bs *BlockFS) SetLast(height base.Height) error {
	bs.Lock()
	defer bs.Unlock()

	return bs.setLast(height)
}

func (bs *BlockFS) setLast(height base.Height) error {
	if height <= base.NilHeight {
		bs.lastINITVoteproof = nil
		bs.lastACCEPTVoteproof = nil

		return nil
	}

	var ivp, avp base.Voteproof
	if vp, err := bs.LoadINITVoteproof(height); err != nil {
		return err
	} else {
		ivp = vp
	}

	if vp, err := bs.LoadACCEPTVoteproof(height); err != nil {
		return err
	} else {
		avp = vp
	}

	bs.lastINITVoteproof = ivp
	bs.lastACCEPTVoteproof = avp

	return nil
}

func (bs *BlockFS) LastVoteproof(stage base.Stage) (base.Voteproof, bool, error) {
	bs.RLock()
	defer bs.RUnlock()

	var vp base.Voteproof
	switch stage {
	case base.StageINIT:
		vp = bs.lastINITVoteproof
	case base.StageACCEPT:
		vp = bs.lastACCEPTVoteproof
	default:
		return nil, false, xerrors.Errorf("invalid stage: %v", stage)
	}

	if vp == nil {
		return nil, false, nil
	}

	return vp, true, nil
}

func (bs *BlockFS) cleanByHeight(height base.Height) error {
	s := height
	for {
		if err := bs.remove(s); err != nil {
			if xerrors.Is(err, NotFoundError) {
				break
			}

			return err
		}

		s++
	}

	return nil
}

func (bs *BlockFS) open(height base.Height, name string) (io.ReadCloser, bool, error) {
	var f string
	if err := bs.walk(bs.base(height), height, nil, func(fp string, fi os.FileInfo) error {
		switch _, _, n, _, _, err := bs.parseFilename(fi.Name()); {
		case err != nil:
			return nil
		case name == n:
			f = fp
			return FoundError.Errorf("found")
		default:
			return nil
		}
	}); err != nil {
		if !xerrors.Is(err, FoundError) {
			return nil, false, err
		}
	}

	var rd io.ReadCloser
	if r, err := bs.fs.Open(f); err != nil {
		return nil, false, err
	} else {
		rd = r
	}

	return rd, strings.HasSuffix(f, ".gz"), nil
}

func (bs *BlockFS) add(height base.Height, bh valuehash.Hash, name string, i interface{}) error {
	bs.fileLock[name].Lock()
	defer bs.fileLock[name].Unlock()

	if b, err := bs.enc.Marshal(i); err != nil {
		return err
	} else if err := bs.save(height, bh, name, b); err != nil {
		err := errors.NewError("failed to save block data, %q", name).Wrap(err)
		if err0 := bs.Cancel(height, bh); err0 != nil {
			return err.Wrap(err0)
		}

		return err
	} else {
		return nil
	}
}

func (bs *BlockFS) load(height base.Height, name string) (interface{}, error) {
	bs.fileLock[name].Lock()
	defer bs.fileLock[name].Unlock()

	r, isCompressed, err := bs.open(height, name)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = r.Close()
	}()

	var rd io.Reader
	switch {
	case !isCompressed:
		rd = r
	default:
		if gr, err := gzip.NewReader(r); err != nil {
			return nil, WrapFSError(err)
		} else {
			rd = gr
		}
	}

	if b, err := ioutil.ReadAll(rd); err != nil {
		return nil, WrapFSError(err)
	} else if hinter, err := bs.enc.DecodeByHint(b); err != nil {
		return nil, err
	} else {
		return hinter, nil
	}
}

func (bs *BlockFS) base(height base.Height) string {
	ht := bs.heightToFilename(height)

	sl := make([]string, 7)
	var i int
	for {
		e := (i * 3) + 3
		if e > len(ht) {
			e = len(ht)
		}

		s := ht[i*3 : e]
		if len(s) < 1 {
			break
		}

		sl[i] = s

		if len(s) < 3 {
			break
		}

		i++
	}

	return filepath.Join("/block/" + strings.Join(sl, "/"))
}

func (bs *BlockFS) prefix(height base.Height, bh valuehash.Hash) string {
	if bh == nil {
		return bs.heightToFilename(height) + "-"
	}

	return fmt.Sprintf("%s-%s-", bs.heightToFilename(height), bh.String())
}

func (bs *BlockFS) heightFromFilename(s string) (base.Height, error) {
	var i string = s

	if strings.Contains(i, "_") {
		i = "-" + strings.ReplaceAll(i, "_", "")
	}

	if h, err := base.NewHeightFromString(i); err != nil {
		return base.NilHeight, xerrors.Errorf("invalid height string: %v, %w", s, err)
	} else {
		return h, nil
	}
}

func (bs *BlockFS) heightToFilename(height base.Height) string {
	h := height.String()
	if height < 0 {
		h = strings.ReplaceAll(h, "-", "_")
	}

	return fmt.Sprintf(heightFormat, h)
}

func (bs *BlockFS) filename(height base.Height, bh valuehash.Hash, fh, name string) string {
	return fmt.Sprintf("%s-%s-%s-%s.jsonld.gz", bs.heightToFilename(height), bh.String(), name, fh)
}

func (bs *BlockFS) parseFilename(s string) (base.Height, valuehash.Hash, string, string, string, error) { // nolint; unparam
	ms := regBlockFilename.FindStringSubmatch(s)
	if n := len(ms); n != 6 {
		return base.NilHeight, nil, "", "", "", xerrors.Errorf("invalid filename string: %v, %d", s, n)
	}

	var height base.Height
	if h, err := bs.heightFromFilename(ms[1]); err != nil {
		return base.NilHeight, nil, "", "", "", xerrors.Errorf("invalid height in filename string: %v, %w", s, err)
	} else {
		height = h
	}

	return height, valuehash.NewBytesFromString(ms[2]), ms[3], ms[4], ms[5], nil
}

func (bs *BlockFS) temp() string {
	return filepath.Join("/tmp", util.UUID().String())
}

func (bs *BlockFS) unstaged(height base.Height, bh valuehash.Hash) string {
	return filepath.Join("/tmp", fmt.Sprintf("%d-%s", height, bh.String()))
}

func (bs *BlockFS) existsWithHash(p string, height base.Height, bh valuehash.Hash) error {
	founds := map[string]bool{}
	for _, f := range blockFiles {
		founds[f] = false
	}

	if err := bs.walk(p, height, bh, func(fp string, fi os.FileInfo) error {
		if _, _, name, _, _, err := bs.parseFilename(fi.Name()); err != nil {
			return nil
		} else if _, found := founds[name]; !found {
			return nil
		} else {
			founds[name] = true
		}

		return nil
	}); err != nil {
		return err
	}

	for _, f := range founds {
		if !f {
			return NotFoundError.Errorf("no block files found")
		}
	}

	return nil
}

func (bs *BlockFS) walk(p string, height base.Height, bh valuehash.Hash, f WalkFunc) error {
	prefix := bs.prefix(height, bh)
	return bs.fs.Walk(p, func(fp string, fi os.FileInfo) error {
		if !strings.HasPrefix(fi.Name(), prefix) {
			return nil
		}

		return f(fp, fi)
	})
}

func (bs *BlockFS) remove(height base.Height) error {
	dir := bs.base(height)
	switch fi, err := bs.fs.Stat(dir); {
	case err != nil:
		return err
	case fi.IsDir():
		if err := bs.fs.RemoveDirectory(dir); err != nil {
			return err
		}
	default:
		if err := bs.fs.Remove(dir); err != nil {
			return err
		}
	}

	return nil
}

func (bs *BlockFS) save(height base.Height, bh valuehash.Hash, name string, b []byte) error {
	var found bool
	for _, i := range blockFiles {
		if name == i {
			found = true

			break
		}
	}

	if !found {
		return xerrors.Errorf("unknown block file found, %q", name)
	}

	// remove previous one
	var founds []string
	if err := bs.walk(bs.unstaged(height, bh), height, bh, func(fp string, fi os.FileInfo) error {
		switch _, _, n, _, _, err := bs.parseFilename(fi.Name()); {
		case err != nil:
			return nil
		case n == name:
			founds = append(founds, fp)

			return nil
		default:
			return nil
		}
	}); err != nil {
		if !xerrors.Is(err, NotFoundError) {
			return err
		}
	}

	for _, f := range founds {
		if err := bs.fs.Remove(f); err != nil {
			if !xerrors.Is(err, NotFoundError) {
				return err
			}
		}
	}

	temp := bs.temp()
	if err := bs.fs.Create(temp, b, true, true); err != nil {
		return err
	}

	f := bs.filename(height, bh, valuehash.SHA256Checksum(b), name)
	unstaged := bs.unstaged(height, bh)
	p := filepath.Join(unstaged, f)

	if err := bs.fs.Remove(p); err != nil {
		if !xerrors.Is(err, NotFoundError) {
			return err
		}
	}

	return bs.fs.Rename(temp, p)
}
