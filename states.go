package mitum

import (
	"sync"
	"time"

	"github.com/spikeekips/mitum/valuehash"
	"golang.org/x/xerrors"
)

type LockedItem struct {
	sync.RWMutex
	value interface{}
}

func NewLockedItem(defaultValue interface{}) *LockedItem {
	return &LockedItem{value: defaultValue}
}

func (li *LockedItem) Value() interface{} {
	li.RLock()
	defer li.RUnlock()

	return li.value
}

func (li *LockedItem) SetValue(value interface{}) *LockedItem {
	li.Lock()
	defer li.Unlock()

	li.value = value

	return li
}

type LocalPolicy struct {
	threshold                      *LockedItem
	timeoutWaitingProposal         *LockedItem
	intervalBroadcastingINITBallot *LockedItem
}

func NewLocalPolicy() *LocalPolicy {
	threshold, _ := NewThreshold(1, 100)
	return &LocalPolicy{
		// NOTE default threshold assumes only one node exists, it means the network is just booted.
		threshold: NewLockedItem(threshold),
		// TODO these values must be reset by last block's data
		timeoutWaitingProposal:         NewLockedItem(time.Second * 3),
		intervalBroadcastingINITBallot: NewLockedItem(time.Second * 1),
	}
}

func (lp *LocalPolicy) Threshold() Threshold {
	return lp.threshold.Value().(Threshold)
}

func (lp *LocalPolicy) SetThreshold(threshold Threshold) *LocalPolicy {
	_ = lp.threshold.SetValue(threshold)

	return lp
}

func (lp *LocalPolicy) TimeoutWaitingProposal() time.Duration {
	return lp.timeoutWaitingProposal.Value().(time.Duration)
}

func (lp *LocalPolicy) SetTimeoutWaitingProposal(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("TimeoutWaitingProposal too short; %v", d)
	}

	_ = lp.timeoutWaitingProposal.SetValue(d)

	return lp, nil
}

func (lp *LocalPolicy) IntervalBroadcastingINITBallot() time.Duration {
	return lp.intervalBroadcastingINITBallot.Value().(time.Duration)
}

func (lp *LocalPolicy) SetIntervalBroadcastingINITBallot(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("IntervalBroadcastingINITBallot too short; %v", d)
	}

	_ = lp.intervalBroadcastingINITBallot.SetValue(d)

	return lp, nil
}

type NodesState struct {
	sync.RWMutex
	nodes map[Address]Node
}

func NewNodesState(nodes []Node) *NodesState {
	m := map[Address]Node{}
	for _, n := range nodes {
		if _, found := m[n.Address()]; found {
			continue
		}
		m[n.Address()] = n
	}

	return &NodesState{nodes: m}
}

func (ns *NodesState) Node(address Address) (Node, bool) {
	ns.RLock()
	defer ns.RUnlock()
	n, found := ns.nodes[address]

	return n, found
}

func (ns *NodesState) Exists(address Address) bool {
	ns.RLock()
	defer ns.RUnlock()

	return ns.exists(address)
}

func (ns *NodesState) exists(address Address) bool {
	_, found := ns.nodes[address]

	return found
}

func (ns *NodesState) Add(nl ...Node) error {
	ns.Lock()
	defer ns.Unlock()

	for _, n := range nl {
		if ns.exists(n.Address()) {
			return xerrors.Errorf("same Address already exists; %v", n.Address())
		}
	}

	for _, n := range nl {
		ns.nodes[n.Address()] = n
	}

	return nil
}

func (ns *NodesState) Remove(addresses ...Address) error {
	ns.Lock()
	defer ns.Unlock()

	for _, address := range addresses {
		if !ns.exists(address) {
			return xerrors.Errorf("Address does not exist; %v", address)
		}
	}

	for _, address := range addresses {
		delete(ns.nodes, address)
	}

	return nil
}

func (ns *NodesState) Len() int {
	return len(ns.nodes)
}

func (ns *NodesState) Traverse(callback func(Node) bool) {
	var nodes []Node
	ns.RLock()
	{
		if len(ns.nodes) < 1 {
			return
		}

		for _, n := range ns.nodes {
			nodes = append(nodes, n)
		}
	}
	ns.RUnlock()

	for _, n := range nodes {
		if !callback(n) {
			break
		}
	}
}

type LocalState struct {
	node                *LocalNode
	policy              *LocalPolicy
	nodes               *NodesState
	lastBlockHeight     *LockedItem // TODO combine these 3 values by last block
	lastBlockHash       *LockedItem
	lastBlockRound      *LockedItem
	lastINITVoteProof   *LockedItem
	lastACCEPTVoteProof *LockedItem
}

func NewLocalState(node *LocalNode, policy *LocalPolicy) *LocalState {
	// TODO fill this values from storage
	return &LocalState{
		node:                node,
		policy:              policy,
		nodes:               NewNodesState(nil),
		lastBlockHash:       NewLockedItem(nil),
		lastBlockHeight:     NewLockedItem(Height(0)),
		lastBlockRound:      NewLockedItem(Round(0)),
		lastINITVoteProof:   NewLockedItem(nil),
		lastACCEPTVoteProof: NewLockedItem(nil),
	}
}

func (ls *LocalState) Node() *LocalNode {
	return ls.node
}

func (ls *LocalState) Policy() *LocalPolicy {
	return ls.policy
}

func (ls *LocalState) Nodes() *NodesState {
	return ls.nodes
}

func (ls *LocalState) LastBlockHash() valuehash.Hash {
	v := ls.lastBlockHash.Value()
	if v == nil {
		return nil
	}

	return v.(valuehash.Hash)
}

func (ls *LocalState) SetLastBlockHash(h valuehash.Hash) *LocalState {
	_ = ls.lastBlockHash.SetValue(h)

	return ls
}

func (ls *LocalState) LastBlockHeight() Height {
	return ls.lastBlockHeight.Value().(Height)
}

func (ls *LocalState) SetLastBlockHeight(height Height) *LocalState {
	_ = ls.lastBlockHeight.SetValue(height)

	return ls
}

func (ls *LocalState) LastBlockRound() Round {
	return ls.lastBlockRound.Value().(Round)
}

func (ls *LocalState) SetLastBlockRound(round Round) *LocalState {
	_ = ls.lastBlockRound.SetValue(round)

	return ls
}

func (ls *LocalState) LastINITVoteProof() VoteProof {
	vp := ls.lastINITVoteProof.Value()
	if vp == nil {
		return nil
	}

	return vp.(VoteProof)
}

func (ls *LocalState) SetLastINITVoteProof(vp VoteProof) *LocalState {
	_ = ls.lastINITVoteProof.SetValue(vp)

	return ls
}

func (ls *LocalState) LastACCEPTVoteProof() VoteProof {
	return ls.lastACCEPTVoteProof.Value().(VoteProof)
}

func (ls *LocalState) SetLastACCEPTVoteProof(vp VoteProof) *LocalState {
	_ = ls.lastACCEPTVoteProof.SetValue(vp)

	return ls
}
