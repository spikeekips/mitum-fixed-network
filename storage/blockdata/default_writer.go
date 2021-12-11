package blockdata

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/tree"
)

var (
	BlockDataWriterType = hint.Type("blockdata-writer")
	BlockDataWriterHint = hint.NewHint(BlockDataWriterType, "v0.0.1")
)

type DefaultWriter struct {
	encoder encoder.Encoder
}

func NewDefaultWriter(enc encoder.Encoder) DefaultWriter {
	return DefaultWriter{encoder: enc}
}

func (DefaultWriter) Hint() hint.Hint {
	return BlockDataWriterHint
}

func (bd DefaultWriter) WriteManifest(w io.Writer, manifest block.Manifest) error {
	return bd.writeItem(w, manifest)
}

func (bd DefaultWriter) WriteOperations(w io.Writer, ops []operation.Operation) error {
	return bd.writeItems(w, ops)
}

func (bd DefaultWriter) WriteOperationsTree(w io.Writer, tr tree.FixedTree) error {
	return bd.writeTree(w, tr)
}

func (bd DefaultWriter) WriteStates(w io.Writer, sts []state.State) error {
	return bd.writeItems(w, sts)
}

func (bd DefaultWriter) WriteStatesTree(w io.Writer, tr tree.FixedTree) error {
	return bd.writeTree(w, tr)
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

func (bd DefaultWriter) WriteProposal(w io.Writer, pr base.SignedBallotFact) error {
	return bd.writeItem(w, pr)
}

func (bd DefaultWriter) ReadManifest(r io.Reader) (block.Manifest, error) {
	b, err := bd.read(r)
	if err != nil {
		return nil, err
	}

	var m block.Manifest
	err = encoder.Decode(b, bd.encoder, &m)
	return m, err
}

func (bd DefaultWriter) ReadOperations(r io.Reader) ([]operation.Operation, error) {
	var ops []operation.Operation

	if err := bd.readItems(
		r,
		func(header ItemsHeader) error {
			ops = make([]operation.Operation, header.Items)

			return nil
		},
		func(index uint64, b []byte) error {
			var op operation.Operation
			if err := encoder.Decode(b, bd.encoder, &op); err != nil {
				return err
			}
			ops[index] = op

			return nil
		},
		300,
	); err != nil {
		return nil, err
	}

	return ops, nil
}

func (bd DefaultWriter) ReadOperationsTree(r io.Reader) (tree.FixedTree, error) {
	return bd.readTree(r, 300)
}

func (bd DefaultWriter) ReadStates(r io.Reader) ([]state.State, error) {
	var sts []state.State

	if err := bd.readItems(
		r,
		func(header ItemsHeader) error {
			sts = make([]state.State, header.Items)

			return nil
		},
		func(index uint64, b []byte) error {
			var st state.State
			if err := encoder.Decode(b, bd.encoder, &st); err != nil {
				return err
			}
			sts[index] = st

			return nil
		},
		300,
	); err != nil {
		return nil, err
	}

	return sts, nil
}

func (bd DefaultWriter) ReadStatesTree(r io.Reader) (tree.FixedTree, error) {
	return bd.readTree(r, 300)
}

func (bd DefaultWriter) ReadINITVoteproof(r io.Reader) (base.Voteproof, error) {
	b, err := bd.read(r)
	if err != nil {
		return nil, err
	}

	var vp base.Voteproof
	err = encoder.Decode(b, bd.encoder, &vp)
	return vp, err
}

func (bd DefaultWriter) ReadACCEPTVoteproof(r io.Reader) (base.Voteproof, error) {
	b, err := bd.read(r)
	if err != nil {
		return nil, err
	}

	var vp base.Voteproof
	err = encoder.Decode(b, bd.encoder, &vp)
	return vp, err
}

func (bd DefaultWriter) ReadSuffrageInfo(r io.Reader) (block.SuffrageInfo, error) {
	b, err := bd.read(r)
	if err != nil {
		return nil, err
	}
	var si block.SuffrageInfo
	err = encoder.Decode(b, bd.encoder, &si)
	return si, err
}

func (bd DefaultWriter) ReadProposal(r io.Reader) (base.SignedBallotFact, error) {
	b, err := bd.read(r)
	if err != nil {
		return nil, err
	}

	var sfs base.SignedBallotFact
	err = encoder.Decode(b, bd.encoder, &sfs)
	return sfs, err
}

func (DefaultWriter) read(r io.Reader) ([]byte, error) {
	i, err := io.ReadAll(r)
	if err != nil {
		return nil, storage.MergeFSError(err)
	}
	return i, nil
}

func (bd DefaultWriter) writeItem(w io.Writer, v interface{}) error {
	if b, err := bd.encoder.Marshal(v); err != nil {
		return err
	} else if _, err := w.Write(b); err != nil {
		return err
	}

	return nil
}

func (bd DefaultWriter) readItems(
	r io.Reader,
	callbackHeader func(ItemsHeader) error,
	callbackItem func(uint64, []byte) error,
	limit int64,
) error {
	return ReadlinesWithIndex(
		r,
		func(b []byte) error {
			var header ItemsHeader
			if err := bd.encoder.Unmarshal(b, &header); err != nil {
				return err
			} else if err := callbackHeader(header); err != nil {
				return err
			} else {
				return nil
			}
		},
		callbackItem,
		limit,
	)
}

func (bd DefaultWriter) writeItems(w io.Writer, v interface{}) error {
	var l reflect.Value

	switch reflect.TypeOf(v).Kind() {
	case reflect.Slice, reflect.Array:
		l = reflect.ValueOf(v)
	default:
		return errors.Errorf("not array or slice: %T", v)
	}

	n := l.Len()
	var index uint64
	return WritelinesWithIndex(
		w,
		func() ([]byte, error) {
			return bd.encoder.Marshal(ItemsHeader{Items: uint64(l.Len())})
		},
		func() (uint64, []byte, error) {
			defer func() {
				index++
			}()

			if n < 1 || index >= uint64(n) {
				return index, nil, io.EOF
			}

			i, err := bd.encoder.Marshal(l.Index(int(index)).Interface())
			if err != nil {
				return index, nil, err
			}
			return index, i, nil
		},
	)
}

func (bd DefaultWriter) writeTree(w io.Writer, tr tree.FixedTree) error {
	if tr.Len() < 1 {
		return nil
	}

	var index uint64
	return WritelinesWithIndex(
		w,
		func() ([]byte, error) {
			return bd.encoder.Marshal(ItemsHeader{Hint: tr.Hint(), Items: uint64(tr.Len())})
		},
		func() (uint64, []byte, error) {
			defer func() {
				index++
			}()

			if i, err := tr.Node(index); err != nil {
				if errors.Is(err, util.NotFoundError) {
					return index, nil, io.EOF
				}

				return index, nil, err
			} else if j, err := bd.encoder.Marshal(i); err != nil {
				return index, nil, err
			} else {
				return index, j, nil
			}
		},
	)
}

func (bd DefaultWriter) readTree(r io.Reader, limit int64) (tree.FixedTree, error) {
	var nodes []tree.FixedTreeNode
	var header ItemsHeader

	if err := ReadlinesWithIndex(
		r,
		func(b []byte) error {
			if err := bd.encoder.Unmarshal(b, &header); err != nil {
				return err
			} else if err := header.Hint.IsCompatible(tree.FixedTreeHint); err != nil {
				return errors.Wrap(err, "unknown FixedTree")
			}

			nodes = make([]tree.FixedTreeNode, header.Items)

			return nil
		},
		func(index uint64, b []byte) error {
			var i tree.FixedTreeNode
			if err := encoder.Decode(b, bd.encoder, &i); err != nil {
				return err
			}
			nodes[index] = i

			return nil
		},
		limit,
	); err != nil {
		return tree.FixedTree{}, err
	}

	if len(nodes) < 1 {
		return tree.EmptyFixedTree(), nil
	}

	return tree.NewFixedTreeWithHint(header.Hint, nodes), nil
}

func WritelinesWithIndex(
	w io.Writer,
	getHeader func() ([]byte, error),
	getItem func() (uint64, []byte, error),
) error {
	if i, err := getHeader(); err != nil {
		return err
	} else if _, err := w.Write(append(i, []byte("\n")...)); err != nil {
		return err
	}

	return util.WritelineAsync(w, func() ([]byte, error) {
		i, j, err := getItem()
		if err != nil {
			return nil, err
		}
		b := []byte(fmt.Sprintf("# index=%d\n", i))

		return append(b, j...), nil
	}, 100)
}

func ReadlinesWithIndex(
	r io.Reader,
	callbackHeader func([]byte) error,
	callbackItem func(uint64, []byte) error,
	limit int64,
) error {
	wk := util.NewErrgroupWorker(context.Background(), limit)
	defer wk.Close()

	errch := make(chan error, 1)
	go func() {
		defer wk.Done()

		var foundHeader bool
		var index uint64

		errch <- util.Readlines(r, func(b []byte) error {
			if !foundHeader {
				if err := callbackHeader(b); err != nil {
					return err
				}
				foundHeader = true

				return nil
			}

			if bytes.HasPrefix(b, []byte("# index=")) {
				a, err := ParseItemIndexLine(b)
				if err != nil {
					return err
				}
				index = a

				return nil
			}

			index := index
			return wk.NewJob(func(context.Context, uint64) error {
				return callbackItem(index, b)
			})
		})
	}()

	err := wk.Wait()
	if cerr := <-errch; cerr != nil {
		return cerr
	}

	return err
}

type ItemsHeader struct {
	Hint  hint.Hint `json:"hint"`
	Items uint64    `json:"items"`
}

func ParseItemIndexLine(b []byte) (uint64, error) {
	if !bytes.HasPrefix(b, []byte("# index=")) {
		return 0, errors.Errorf("invalid item index, %q", string(b))
	}

	var i uint64
	switch n, err := fmt.Sscanf(string(b), "# index=%d", &i); {
	case err != nil:
		return 0, errors.Wrap(err, "invalid item index")
	case n != 1:
		return 0, errors.Errorf("invalid item index, %d", n)
	default:
		return i, nil
	}
}
