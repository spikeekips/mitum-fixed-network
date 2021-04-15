package blockdata

import (
	"bufio"
	"io"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/tree"
	"golang.org/x/xerrors"
)

var (
	BlockDataWriterType           = hint.MustNewType(0x01, 0x92, "blockdata-writer")
	BlockDataWriterHint hint.Hint = hint.MustHint(BlockDataWriterType, "0.0.1")
)

type DefaultWriter struct {
	encoder encoder.Encoder
}

func NewDefaultWriter(encoder encoder.Encoder) DefaultWriter {
	return DefaultWriter{encoder: encoder}
}

func (bd DefaultWriter) Hint() hint.Hint {
	return BlockDataWriterHint
}

func (bd DefaultWriter) WriteManifest(w io.Writer, manifest block.Manifest) error {
	return bd.writeItem(w, manifest)
}

func (bd DefaultWriter) WriteOperations(w io.Writer, ops []operation.Operation) error {
	var i int = -1
	return bd.writeItems(w, func() (interface{}, error) {
		i++

		if n := len(ops); n < 1 {
			return nil, io.EOF
		} else if i >= n {
			return nil, io.EOF
		}

		return ops[i], nil
	})
}

func (bd DefaultWriter) WriteOperationsTree(w io.Writer, tr tree.FixedTree) error {
	return bd.writeItem(w, tr)
}

func (bd DefaultWriter) WriteStates(w io.Writer, sts []state.State) error {
	var i int = -1
	return bd.writeItems(w, func() (interface{}, error) {
		i++

		if n := len(sts); n < 1 {
			return nil, io.EOF
		} else if i >= n {
			return nil, io.EOF
		}

		return sts[i], nil
	})
}

func (bd DefaultWriter) WriteStatesTree(w io.Writer, tr tree.FixedTree) error {
	return bd.writeItem(w, tr)
}

func (bd DefaultWriter) WriteINITVoteproof(w io.Writer, vp base.Voteproof) error {
	return bd.writeItem(w, vp)
}

func (bd DefaultWriter) WriteACCEPTVoteproof(w io.Writer, vp base.Voteproof) error {
	return bd.writeItem(w, vp)
}

func (bd DefaultWriter) WriteSuffrageInfo(w io.Writer, si block.SuffrageInfo) error {
	return bd.writeItem(w, si)
}

func (bd DefaultWriter) WriteProposal(w io.Writer, pr ballot.Proposal) error {
	return bd.writeItem(w, pr)
}

func (bd DefaultWriter) ReadManifest(r io.Reader) (block.Manifest, error) {
	if b, err := bd.read(r); err != nil {
		return nil, err
	} else {
		return block.DecodeManifest(bd.encoder, b)
	}
}

func (bd DefaultWriter) ReadOperations(r io.Reader) ([]operation.Operation, error) {
	var ops []operation.Operation

	if err := bd.readlines(r, func(b []byte) error {
		if i, err := operation.DecodeOperation(bd.encoder, b); err != nil {
			return err
		} else {
			ops = append(ops, i)

			return nil
		}
	}); err != nil {
		return nil, err
	}

	return ops, nil
}

func (bd DefaultWriter) ReadOperationsTree(r io.Reader) (tree.FixedTree, error) {
	if b, err := bd.read(r); err != nil {
		return tree.FixedTree{}, err
	} else {
		return tree.DecodeFixedTree(bd.encoder, b)
	}
}

func (bd DefaultWriter) ReadStates(r io.Reader) ([]state.State, error) {
	var sts []state.State

	if err := bd.readlines(r, func(b []byte) error {
		if i, err := state.DecodeState(bd.encoder, b); err != nil {
			return err
		} else {
			sts = append(sts, i)

			return nil
		}
	}); err != nil {
		return nil, err
	}

	return sts, nil
}

func (bd DefaultWriter) ReadStatesTree(r io.Reader) (tree.FixedTree, error) {
	if b, err := bd.read(r); err != nil {
		return tree.FixedTree{}, err
	} else {
		return tree.DecodeFixedTree(bd.encoder, b)
	}
}

func (bd DefaultWriter) ReadINITVoteproof(r io.Reader) (base.Voteproof, error) {
	if b, err := bd.read(r); err != nil {
		return nil, err
	} else {
		return base.DecodeVoteproof(bd.encoder, b)
	}
}

func (bd DefaultWriter) ReadACCEPTVoteproof(r io.Reader) (base.Voteproof, error) {
	if b, err := bd.read(r); err != nil {
		return nil, err
	} else {
		return base.DecodeVoteproof(bd.encoder, b)
	}
}

func (bd DefaultWriter) ReadSuffrageInfo(r io.Reader) (block.SuffrageInfo, error) {
	if b, err := bd.read(r); err != nil {
		return nil, err
	} else {
		return block.DecodeSuffrageInfo(bd.encoder, b)
	}
}

func (bd DefaultWriter) ReadProposal(r io.Reader) (ballot.Proposal, error) {
	if b, err := bd.read(r); err != nil {
		return nil, err
	} else {
		return ballot.DecodeProposal(bd.encoder, b)
	}
}

func (bd DefaultWriter) read(r io.Reader) ([]byte, error) {
	if i, err := io.ReadAll(r); err != nil {
		return nil, storage.WrapFSError(err)
	} else {
		return i, nil
	}
}

func (bd DefaultWriter) readlines(r io.Reader, callback func([]byte) error) error {
	return Readlines(r, callback)
}

func (bd DefaultWriter) writeItem(w io.Writer, v interface{}) error {
	if b, err := bd.encoder.Marshal(v); err != nil {
		return err
	} else if _, err := w.Write(b); err != nil {
		return err
	}

	return nil
}

func (bd DefaultWriter) writeItems(w io.Writer, get func() (interface{}, error)) error {
	return Writeline(w, func() ([]byte, error) {
		if i, err := get(); err != nil {
			return nil, err
		} else {
			return bd.encoder.Marshal(i)
		}
	})
}

func Writeline(w io.Writer, get func() ([]byte, error)) error {
	for {
		if i, err := get(); err != nil {
			if xerrors.Is(err, io.EOF) {
				break
			}

			return err
		} else if _, err := w.Write(append(i, []byte("\n")...)); err != nil {
			return err
		}
	}

	return nil
}

func Readlines(r io.Reader, callback func([]byte) error) error {
	br := bufio.NewReader(r)
	for {
		l, err := br.ReadBytes('\n')
		if err != nil {
			if xerrors.Is(err, io.EOF) {
				break
			}

			return err
		}

		if err := callback(l); err != nil {
			return err
		}
	}

	return nil
}
