package operation

import (
	"fmt"

	"github.com/spikeekips/mitum/util/errors"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/tree"
	"golang.org/x/xerrors"
)

var (
	FixedTreeNodeType   = hint.Type("operation-fixedtree-node")
	FixedTreeNodeHint   = hint.NewHint(FixedTreeNodeType, "v0.0.1")
	BaseReasonErrorType = hint.Type("base-operation-reason")
	BaseReasonErrorHint = hint.NewHint(BaseReasonErrorType, "v0.0.1")
)

type FixedTreeNode struct {
	tree.BaseFixedTreeNode
	inState bool
	reason  ReasonError
}

func NewFixedTreeNode(index uint64, key []byte, inState bool, reason error) FixedTreeNode {
	return NewFixedTreeNodeWithHash(index, key, nil, inState, reason)
}

func NewFixedTreeNodeWithHash(index uint64, key, hash []byte, inState bool, reason error) FixedTreeNode {
	var operr ReasonError
	if reason != nil {
		operr = NewBaseReasonErrorFromError(reason)
	}

	return FixedTreeNode{
		BaseFixedTreeNode: tree.NewBaseFixedTreeNodeWithHash(index, key, hash),
		inState:           inState,
		reason:            operr,
	}
}

func (FixedTreeNode) Hint() hint.Hint {
	return FixedTreeNodeHint
}

func (no FixedTreeNode) InState() bool {
	return no.inState
}

func (no FixedTreeNode) Reason() ReasonError {
	return no.reason
}

func (no FixedTreeNode) SetHash(h []byte) tree.FixedTreeNode {
	no.BaseFixedTreeNode = no.BaseFixedTreeNode.SetHash(h).(tree.BaseFixedTreeNode)

	return no
}

func (no FixedTreeNode) Equal(n tree.FixedTreeNode) bool {
	if !no.BaseFixedTreeNode.Equal(n) {
		return false
	}

	nno, ok := n.(FixedTreeNode)
	if !ok {
		return true
	}

	switch {
	case no.inState != nno.inState:
		return false
	default:
		return true
	}
}

type ReasonError interface {
	error
	hint.Hinter
	Msg() string
	Data() map[string]interface{}
	Errorf(string, ...interface{}) ReasonError
	Wrap(error) ReasonError
}

type BaseReasonError struct {
	*errors.NError
	msg  string
	data map[string]interface{}
}

func NewBaseReasonErrorFromError(err error) ReasonError {
	var operr ReasonError
	if xerrors.As(err, &operr) {
		return operr
	}

	var nerror *errors.NError
	if !xerrors.As(err, &nerror) {
		nerror = errors.NewError("").Wrap(err).SetFrame(2)
	}

	return BaseReasonError{NError: nerror}
}

func NewBaseReasonError(s string, a ...interface{}) BaseReasonError {
	return BaseReasonError{NError: errors.NewError("").Errorf(s, a...).SetFrame(2)}
}

func (BaseReasonError) Hint() hint.Hint {
	return BaseReasonErrorHint
}

func (e BaseReasonError) Error() string {
	if e.NError == nil {
		return e.msg
	}

	return e.NError.Error()
}

func (e BaseReasonError) Format(s fmt.State, v rune) {
	if e.NError != nil {
		e.NError.Format(s, v)
	}
}

func (e BaseReasonError) Msg() string {
	if len(e.msg) > 0 {
		return e.msg
	}

	if e.NError == nil {
		return ""
	} else if e.Err() == nil {
		return e.NError.Msg()
	}

	var m string
	if s := e.NError.Msg(); len(s) > 0 {
		m = s + "; "
	}
	return fmt.Sprintf("%s%v", m, e.Err())
}

func (e BaseReasonError) Data() map[string]interface{} {
	return e.data
}

func (e BaseReasonError) SetData(data map[string]interface{}) BaseReasonError {
	e.data = data

	return e
}

func (e BaseReasonError) Wrap(err error) ReasonError {
	var operr BaseReasonError
	if xerrors.As(err, &operr) {
		return operr
	}

	return BaseReasonError{NError: e.NError.Wrap(err), data: e.data}
}

func (e BaseReasonError) Errorf(s string, a ...interface{}) ReasonError {
	return BaseReasonError{NError: e.NError.Errorf(s, a...), data: e.data}
}
