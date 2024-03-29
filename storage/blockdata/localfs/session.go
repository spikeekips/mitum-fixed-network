package localfs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/tree"
)

var (
	DefaultFilePermission      os.FileMode = 0o644
	DefaultDirectoryPermission os.FileMode = 0o755
)

type Session struct {
	sync.RWMutex
	locks            map[string]*sync.RWMutex
	height           base.Height
	root             string
	writer           blockdata.Writer
	operationsWriter io.WriteCloser
	statesWriter     io.WriteCloser
	mapData          block.BaseBlockdataMap
}

func NewSession(root string, writer blockdata.Writer, height base.Height) (*Session, error) {
	if fi, err := os.Stat(root); err != nil {
		return nil, storage.MergeFSError(err)
	} else if !fi.IsDir() {
		return nil, storage.FSError.Errorf("session root, %q is not directory", root)
	}

	return &Session{
		locks: map[string]*sync.RWMutex{
			block.BlockdataManifest:        {},
			block.BlockdataOperations:      {},
			block.BlockdataOperationsTree:  {},
			block.BlockdataStates:          {},
			block.BlockdataStatesTree:      {},
			block.BlockdataINITVoteproof:   {},
			block.BlockdataACCEPTVoteproof: {},
			block.BlockdataSuffrageInfo:    {},
			block.BlockdataProposal:        {},
		},
		height:  height,
		root:    root,
		writer:  writer,
		mapData: block.NewBaseBlockdataMap(writer.Hint(), height),
	}, nil
}

func (ss *Session) Height() base.Height {
	return ss.height
}

func (ss *Session) SetManifest(manifest block.Manifest) error {
	ss.locks[block.BlockdataManifest].Lock()
	defer ss.locks[block.BlockdataManifest].Unlock()

	if i, ok := manifest.(block.Block); ok {
		manifest = i.Manifest()
	}

	if err := ss.writeAndClose(block.BlockdataManifest, func(w io.Writer) error {
		return ss.writer.WriteManifest(w, manifest)
	}); err != nil {
		return err
	}
	ss.Lock()
	ss.mapData = ss.mapData.SetBlock(manifest.Hash())
	ss.Unlock()

	return nil
}

func (ss *Session) AddOperations(ops ...operation.Operation) error {
	ss.locks[block.BlockdataOperations].Lock()
	defer ss.locks[block.BlockdataOperations].Unlock()

	if ss.operationsWriter == nil {
		i, err := ss.newWriter(block.BlockdataOperations)
		if err != nil {
			return err
		}
		ss.operationsWriter = i
	}

	return ss.writer.WriteOperations(ss.operationsWriter, ops)
}

func (ss *Session) CloseOperations() error {
	ss.locks[block.BlockdataOperations].Lock()
	defer ss.locks[block.BlockdataOperations].Unlock()

	if ss.operationsWriter == nil {
		return nil
	}

	if err := ss.operationsWriter.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		return err
	}
	ss.operationsWriter = nil

	return ss.setToMapData(block.BlockdataOperations)
}

func (ss *Session) SetOperationsTree(ft tree.FixedTree) error {
	ss.locks[block.BlockdataOperationsTree].Lock()
	defer ss.locks[block.BlockdataOperationsTree].Unlock()

	return ss.writeAndClose(block.BlockdataOperationsTree, func(w io.Writer) error {
		return ss.writer.WriteOperationsTree(w, ft)
	})
}

func (ss *Session) AddStates(sts ...state.State) error {
	ss.locks[block.BlockdataStates].Lock()
	defer ss.locks[block.BlockdataStates].Unlock()

	if ss.statesWriter == nil {
		i, err := ss.newWriter(block.BlockdataStates)
		if err != nil {
			return err
		}
		ss.statesWriter = i
	}

	return ss.writer.WriteStates(ss.statesWriter, sts)
}

func (ss *Session) CloseStates() error {
	ss.locks[block.BlockdataStates].Lock()
	defer ss.locks[block.BlockdataStates].Unlock()

	if ss.statesWriter == nil {
		return nil
	}

	if err := ss.statesWriter.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		return err
	}
	ss.statesWriter = nil

	return ss.setToMapData(block.BlockdataStates)
}

func (ss *Session) SetStatesTree(ft tree.FixedTree) error {
	ss.locks[block.BlockdataStatesTree].Lock()
	defer ss.locks[block.BlockdataStatesTree].Unlock()

	return ss.writeAndClose(block.BlockdataStatesTree, func(w io.Writer) error {
		return ss.writer.WriteStatesTree(w, ft)
	})
}

func (ss *Session) SetINITVoteproof(voteproof base.Voteproof) error {
	if voteproof != nil {
		if voteproof.Stage() != base.StageINIT {
			return errors.Errorf("not init voteproof")
		}
	}

	ss.locks[block.BlockdataINITVoteproof].Lock()
	defer ss.locks[block.BlockdataINITVoteproof].Unlock()

	return ss.writeAndClose(block.BlockdataINITVoteproof, func(w io.Writer) error {
		return ss.writer.WriteINITVoteproof(w, voteproof)
	})
}

func (ss *Session) SetACCEPTVoteproof(voteproof base.Voteproof) error {
	if voteproof != nil {
		if voteproof.Stage() != base.StageACCEPT {
			return errors.Errorf("not accept voteproof")
		}
	}

	ss.locks[block.BlockdataACCEPTVoteproof].Lock()
	defer ss.locks[block.BlockdataACCEPTVoteproof].Unlock()

	return ss.writeAndClose(block.BlockdataACCEPTVoteproof, func(w io.Writer) error {
		return ss.writer.WriteACCEPTVoteproof(w, voteproof)
	})
}

func (ss *Session) SetSuffrageInfo(suffrageInfo block.SuffrageInfo) error {
	ss.locks[block.BlockdataSuffrageInfo].Lock()
	defer ss.locks[block.BlockdataSuffrageInfo].Unlock()

	return ss.writeAndClose(block.BlockdataSuffrageInfo, func(w io.Writer) error {
		return ss.writer.WriteSuffrageInfo(w, suffrageInfo)
	})
}

func (ss *Session) SetProposal(sfs base.SignedBallotFact) error {
	ss.locks[block.BlockdataProposal].Lock()
	defer ss.locks[block.BlockdataProposal].Unlock()

	return ss.writeAndClose(block.BlockdataProposal, func(w io.Writer) error {
		return ss.writer.WriteProposal(w, sfs)
	})
}

func (ss *Session) SetBlock(blk block.Block) error {
	var initVoteproof, acceptVoteproof base.Voteproof
	if vp := blk.ConsensusInfo().INITVoteproof(); vp != nil {
		initVoteproof = vp
	}
	if vp := blk.ConsensusInfo().ACCEPTVoteproof(); vp != nil {
		acceptVoteproof = vp
	}

	funcs := []func() error{
		func() error { return ss.SetManifest(blk.Manifest()) },
		func() error {
			if err := ss.AddOperations(blk.Operations()...); err != nil {
				return err
			}

			return ss.CloseOperations()
		},
		func() error { return ss.SetOperationsTree(blk.OperationsTree()) },
		func() error {
			if err := ss.AddStates(blk.States()...); err != nil {
				return err
			}

			return ss.CloseStates()
		},
		func() error { return ss.SetStatesTree(blk.StatesTree()) },
		func() error { return ss.SetINITVoteproof(initVoteproof) },
		func() error { return ss.SetACCEPTVoteproof(acceptVoteproof) },
		func() error { return ss.SetSuffrageInfo(blk.ConsensusInfo().SuffrageInfo()) },
		func() error { return ss.SetProposal(blk.ConsensusInfo().Proposal()) },
	}

	for i := range funcs {
		if err := funcs[i](); err != nil {
			return err
		}
	}

	return nil
}

func (ss *Session) done() (block.BaseBlockdataMap, error) {
	ss.Lock()
	defer ss.Unlock()

	// NOTE check mapData
	if err := ss.mapData.IsReadyToHash(); err != nil {
		return block.BaseBlockdataMap{}, err
	}

	mapData, err := ss.mapData.UpdateHash()
	if err != nil {
		return block.BaseBlockdataMap{}, err
	}

	if err := mapData.IsValid(nil); err != nil {
		return block.BaseBlockdataMap{}, err
	} else if err := mapData.Exists("/"); err != nil {
		return block.BaseBlockdataMap{}, err
	}

	return mapData, nil
}

func (ss *Session) Cancel() error {
	return ss.clean()
}

func (ss *Session) Import(dataType string, r io.Reader) (string, error) {
	ss.locks[dataType].Lock()
	defer ss.locks[dataType].Unlock()

	w, err := ss.newWriter(dataType)
	if err != nil {
		return "", err
	}

	if err := func() error {
		defer func() {
			_ = w.Close()
		}()

		_, err := io.Copy(w, r)

		return err
	}(); err != nil {
		return "", err
	}

	return ss.setToMapDataWithFilename(dataType)
}

func (ss *Session) tempPath(dataType string) string {
	return filepath.Join(ss.root, fmt.Sprintf(".%s.gz", dataType))
}

func (ss *Session) newWriter(dataType string) (io.WriteCloser, error) {
	i, err := os.OpenFile(
		filepath.Clean(ss.tempPath(dataType)),
		os.O_CREATE|os.O_WRONLY,
		DefaultFilePermission,
	)
	if err != nil {
		return nil, storage.MergeFSError(err)
	}
	return util.NewGzipWriter(i), nil
}

func (ss *Session) writeAndClose(dataType string, writer func(io.Writer) error) error {
	w, err := ss.newWriter(dataType)
	if err != nil {
		return err
	}

	err = writer(w)
	_ = w.Close()
	if err != nil {
		return err
	}

	return ss.setToMapData(dataType)
}

func (ss *Session) clean() error {
	if err := os.RemoveAll(ss.root); err != nil {
		return storage.MergeFSError(err)
	}

	return nil
}

func (ss *Session) setToMapData(dataType string) error {
	_, err := ss.setToMapDataWithFilename(dataType)

	return err
}

func (ss *Session) setToMapDataWithFilename(dataType string) (string, error) {
	ss.Lock()
	defer ss.Unlock()

	p := ss.tempPath(dataType)
	if fi, err := os.Stat(p); err != nil {
		return "", storage.MergeFSError(err)
	} else if fi.IsDir() {
		return "", storage.FSError.Errorf("temp path, %q is directory", p)
	}

	checksum, err := util.GenerateFileChecksum(p)
	if err != nil {
		return "", storage.MergeFSError(err)
	}

	t := filepath.Join(filepath.Dir(p), fmt.Sprintf(BlockFileFormats, ss.height, dataType, checksum))

	if err := os.Rename(p, t); err != nil {
		return "", storage.MergeFSError(err)
	}

	item := block.NewBaseBlockdataMapItem(dataType, checksum, "file://"+t)
	if err := item.IsValid(nil); err != nil {
		return "", err
	} else if i, err := ss.mapData.SetItem(item); err != nil {
		return "", err
	} else {
		ss.mapData = i
	}

	return t, nil
}
