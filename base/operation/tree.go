package operation

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/tree"
)

var (
	FixedTreeNodeType   = hint.Type("operation-fixedtree-node")
	FixedTreeNodeHint   = hint.NewHint(FixedTreeNodeType, "v0.0.1")
	FixedTreeNodeHinter = FixedTreeNode{
		BaseFixedTreeNode: tree.BaseFixedTreeNode{BaseHinter: hint.NewBaseHinter(FixedTreeNodeHint)},
	}
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
		BaseFixedTreeNode: tree.NewBaseFixedTreeNodeWithHash(FixedTreeNodeHint, index, key, hash),
		inState:           inState,
		reason:            operr,
	}
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
}

type BaseReasonError struct {
	*util.NError
	msg  string
	data map[string]interface{}
}

func NewBaseReasonErrorFromError(err error) ReasonError {
	var operr ReasonError
	if errors.As(err, &operr) {
		return operr
	}

	var nerr *util.NError
	if !errors.As(err, &nerr) {
		nerr = util.NewError("").Merge(err)
	}

	return BaseReasonError{NError: nerr}
}

func NewBaseReasonError(s string, a ...interface{}) BaseReasonError {
	return BaseReasonError{NError: util.NewError("").Errorf(s, a...).Caller(2)}
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
	}

	return e.NError.Error()
}

func (e BaseReasonError) Data() map[string]interface{} {
	return e.data
}

func (e BaseReasonError) SetData(data map[string]interface{}) BaseReasonError {
	e.data = data

	return e
}

func (e BaseReasonError) Errorf(s string, a ...interface{}) ReasonError {
	return BaseReasonError{
		NError: e.NError.Errorf(s, a...),
		msg:    e.msg,
		data:   e.data,
	}
}
