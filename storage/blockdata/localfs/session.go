package localfs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/tree"
	"golang.org/x/xerrors"
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
	mapData          block.BaseBlockDataMap
}

func NewSession(root string, writer blockdata.Writer, height base.Height) (*Session, error) {
	if fi, err := os.Stat(root); err != nil {
		return nil, storage.WrapFSError(err)
	} else if !fi.IsDir() {
		return nil, storage.FSError.Errorf("session root, %q is not directory", root)
	}

	return &Session{
		locks: map[string]*sync.RWMutex{
			block.BlockDataManifest:        {},
			block.BlockDataOperations:      {},
			block.BlockDataOperationsTree:  {},
			block.BlockDataStates:          {},
			block.BlockDataStatesTree:      {},
			block.BlockDataINITVoteproof:   {},
			block.BlockDataACCEPTVoteproof: {},
			block.BlockDataSuffrageInfo:    {},
			block.BlockDataProposal:        {},
		},
		height:  height,
		root:    root,
		writer:  writer,
		mapData: block.NewBaseBlockDataMap(writer.Hint(), height),
	}, nil
}

func (ss *Session) Height() base.Height {
	return ss.height
}

func (ss *Session) SetManifest(manifest block.Manifest) error {
	ss.locks[block.BlockDataManifest].Lock()
	defer ss.locks[block.BlockDataManifest].Unlock()

	if i, ok := manifest.(block.Block); ok {
		manifest = i.Manifest()
	}

	if err := ss.writeAndClose(block.BlockDataManifest, func(w io.Writer) error {
		return ss.writer.WriteManifest(w, manifest)
	}); err != nil {
		return err
	} else {
		ss.Lock()
		ss.mapData = ss.mapData.SetBlock(manifest.Hash())
		ss.Unlock()

		return nil
	}
}

func (ss *Session) AddOperations(ops ...operation.Operation) error {
	ss.locks[block.BlockDataOperations].Lock()
	defer ss.locks[block.BlockDataOperations].Unlock()

	if ss.operationsWriter == nil {
		if i, err := ss.newWriter(block.BlockDataOperations); err != nil {
			return err
		} else {
			ss.operationsWriter = i
		}
	}

	return ss.writer.WriteOperations(ss.operationsWriter, ops)
}

func (ss *Session) CloseOperations() error {
	ss.locks[block.BlockDataOperations].Lock()
	defer ss.locks[block.BlockDataOperations].Unlock()

	if ss.operationsWriter == nil {
		return nil
	}

	if err := ss.operationsWriter.Close(); err != nil && !xerrors.Is(err, os.ErrClosed) {
		return err
	} else {
		ss.operationsWriter = nil
	}

	return ss.setToMapData(block.BlockDataOperations)
}

func (ss *Session) SetOperationsTree(ft tree.FixedTree) error {
	ss.locks[block.BlockDataOperationsTree].Lock()
	defer ss.locks[block.BlockDataOperationsTree].Unlock()

	return ss.writeAndClose(block.BlockDataOperationsTree, func(w io.Writer) error {
		return ss.writer.WriteOperationsTree(w, ft)
	})
}

func (ss *Session) AddStates(sts ...state.State) error {
	ss.locks[block.BlockDataStates].Lock()
	defer ss.locks[block.BlockDataStates].Unlock()

	if ss.statesWriter == nil {
		if i, err := ss.newWriter(block.BlockDataStates); err != nil {
			return err
		} else {
			ss.statesWriter = i
		}
	}

	return ss.writer.WriteStates(ss.statesWriter, sts)
}

func (ss *Session) CloseStates() error {
	ss.locks[block.BlockDataStates].Lock()
	defer ss.locks[block.BlockDataStates].Unlock()

	if ss.statesWriter == nil {
		return nil
	}

	if err := ss.statesWriter.Close(); err != nil && !xerrors.Is(err, os.ErrClosed) {
		return err
	} else {
		ss.statesWriter = nil
	}

	return ss.setToMapData(block.BlockDataStates)
}

func (ss *Session) SetStatesTree(ft tree.FixedTree) error {
	ss.locks[block.BlockDataStatesTree].Lock()
	defer ss.locks[block.BlockDataStatesTree].Unlock()

	return ss.writeAndClose(block.BlockDataStatesTree, func(w io.Writer) error {
		return ss.writer.WriteStatesTree(w, ft)
	})
}

func (ss *Session) SetINITVoteproof(voteproof base.Voteproof) error {
	if voteproof != nil {
		if voteproof.Stage() != base.StageINIT {
			return xerrors.Errorf("not init voteproof")
		}
	}

	ss.locks[block.BlockDataINITVoteproof].Lock()
	defer ss.locks[block.BlockDataINITVoteproof].Unlock()

	return ss.writeAndClose(block.BlockDataINITVoteproof, func(w io.Writer) error {
		return ss.writer.WriteINITVoteproof(w, voteproof)
	})
}

func (ss *Session) SetACCEPTVoteproof(voteproof base.Voteproof) error {
	if voteproof != nil {
		if voteproof.Stage() != base.StageACCEPT {
			return xerrors.Errorf("not accept voteproof")
		}
	}

	ss.locks[block.BlockDataACCEPTVoteproof].Lock()
	defer ss.locks[block.BlockDataACCEPTVoteproof].Unlock()

	return ss.writeAndClose(block.BlockDataACCEPTVoteproof, func(w io.Writer) error {
		return ss.writer.WriteACCEPTVoteproof(w, voteproof)
	})
}

func (ss *Session) SetSuffrageInfo(suffrageInfo block.SuffrageInfo) error {
	ss.locks[block.BlockDataSuffrageInfo].Lock()
	defer ss.locks[block.BlockDataSuffrageInfo].Unlock()

	return ss.writeAndClose(block.BlockDataSuffrageInfo, func(w io.Writer) error {
		return ss.writer.WriteSuffrageInfo(w, suffrageInfo)
	})
}

func (ss *Session) SetProposal(proposal ballot.Proposal) error {
	ss.locks[block.BlockDataProposal].Lock()
	defer ss.locks[block.BlockDataProposal].Unlock()

	return ss.writeAndClose(block.BlockDataProposal, func(w io.Writer) error {
		return ss.writer.WriteProposal(w, proposal)
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

func (ss *Session) done() (block.BaseBlockDataMap, error) {
	ss.Lock()
	defer ss.Unlock()

	// NOTE check mapData
	if err := ss.mapData.IsReadyToHash(); err != nil {
		return block.BaseBlockDataMap{}, err
	}

	mapData := ss.mapData
	if i, err := mapData.UpdateHash(); err != nil {
		return block.BaseBlockDataMap{}, err
	} else {
		mapData = i
	}

	if err := mapData.IsValid(nil); err != nil {
		return block.BaseBlockDataMap{}, err
	} else if err := mapData.Exists("/"); err != nil {
		return block.BaseBlockDataMap{}, err
	}

	return mapData, nil
}

func (ss *Session) Cancel() error {
	if err := ss.clean(); err != nil {
		return err
	}

	return nil
}

func (ss *Session) Import(dataType string, r io.Reader) error {
	ss.locks[dataType].Lock()
	defer ss.locks[dataType].Unlock()

	return ss.writeAndClose(dataType, func(w io.Writer) error {
		_, err := io.Copy(w, r)

		return storage.WrapFSError(err)
	})
}

func (ss *Session) tempPath(dataType string) string {
	return filepath.Join(ss.root, fmt.Sprintf(".%s.gz", dataType))
}

func (ss *Session) newWriter(dataType string) (io.WriteCloser, error) {
	if i, err := os.OpenFile(
		filepath.Clean(ss.tempPath(dataType)),
		os.O_CREATE|os.O_WRONLY,
		DefaultFilePermission,
	); err != nil {
		return nil, storage.WrapFSError(err)
	} else {
		return util.NewGzipWriter(i), nil
	}
}

func (ss *Session) writeAndClose(dataType string, writer func(io.Writer) error) error {
	var w io.WriteCloser
	if i, err := ss.newWriter(dataType); err != nil {
		return err
	} else {
		w = i
	}

	err := writer(w)
	_ = w.Close()
	if err != nil {
		return err
	}

	return ss.setToMapData(dataType)
}

func (ss *Session) clean() error {
	if err := os.RemoveAll(ss.root); err != nil {
		return storage.WrapFSError(err)
	}

	return nil
}

func (ss *Session) setToMapData(dataType string) error {
	ss.Lock()
	defer ss.Unlock()

	p := ss.tempPath(dataType)
	if fi, err := os.Stat(p); err != nil {
		return storage.WrapFSError(err)
	} else if fi.IsDir() {
		return storage.FSError.Errorf("temp path, %q is directory", p)
	}

	var checksum string
	if i, err := util.GenerateFileChecksum(p); err != nil {
		return storage.WrapFSError(err)
	} else {
		checksum = i
	}

	t := filepath.Join(filepath.Dir(p), fmt.Sprintf(BlockFileFormats, ss.height, dataType, checksum))

	if err := os.Rename(p, t); err != nil {
		return storage.WrapFSError(err)
	}

	item := block.NewBaseBlockDataMapItem(dataType, checksum, "file://"+t)
	if err := item.IsValid(nil); err != nil {
		return err
	} else if i, err := ss.mapData.SetItem(item); err != nil {
		return err
	} else {
		ss.mapData = i
	}

	return nil
}
