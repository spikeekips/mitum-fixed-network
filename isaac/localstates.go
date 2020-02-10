package isaac

import (
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
)

type LocalPolicy struct {
	threshold                        *util.LockedItem
	timeoutWaitingProposal           *util.LockedItem
	intervalBroadcastingINITBallot   *util.LockedItem
	waitBroadcastingACCEPTBallot     *util.LockedItem
	intervalBroadcastingACCEPTBallot *util.LockedItem
	numberOfActingSuffrageNodes      *util.LockedItem
}

func NewLocalPolicy() *LocalPolicy {
	threshold, _ := NewThreshold(1, 100)
	return &LocalPolicy{
		// NOTE default threshold assumes only one node exists, it means the network is just booted.
		threshold: util.NewLockedItem(threshold),
		// TODO these values must be reset by last block's data
		timeoutWaitingProposal:           util.NewLockedItem(time.Second * 5),
		intervalBroadcastingINITBallot:   util.NewLockedItem(time.Second * 1),
		waitBroadcastingACCEPTBallot:     util.NewLockedItem(time.Second * 2),
		intervalBroadcastingACCEPTBallot: util.NewLockedItem(time.Second * 1),
		numberOfActingSuffrageNodes:      util.NewLockedItem(uint(1)),
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

func (lp *LocalPolicy) WaitBroadcastingACCEPTBallot() time.Duration {
	return lp.waitBroadcastingACCEPTBallot.Value().(time.Duration)
}

func (lp *LocalPolicy) SetWaitBroadcastingACCEPTBallot(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("WaitBroadcastingACCEPTBallot too short; %v", d)
	}

	_ = lp.waitBroadcastingACCEPTBallot.SetValue(d)

	return lp, nil
}

func (lp *LocalPolicy) IntervalBroadcastingACCEPTBallot() time.Duration {
	return lp.intervalBroadcastingACCEPTBallot.Value().(time.Duration)
}

func (lp *LocalPolicy) SetIntervalBroadcastingACCEPTBallot(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("IntervalBroadcastingACCEPTBallot too short; %v", d)
	}

	_ = lp.intervalBroadcastingACCEPTBallot.SetValue(d)

	return lp, nil
}

func (lp *LocalPolicy) NumberOfActingSuffrageNodes() uint {
	return lp.numberOfActingSuffrageNodes.Value().(uint)
}

func (lp *LocalPolicy) SetNumberOfActingSuffrageNodes(i uint) (*LocalPolicy, error) {
	if i < 1 {
		return nil, xerrors.Errorf("NumberOfActingSuffrageNodes should be greater than 0; %v", i)
	}

	_ = lp.numberOfActingSuffrageNodes.SetValue(i)

	return lp, nil
}

type NodesState struct {
	sync.RWMutex
	localNode *LocalNode
	nodes     map[Address]Node
}

func NewNodesState(localNode *LocalNode, nodes []Node) *NodesState {
	m := map[Address]Node{}
	for _, n := range nodes {
		if n.Address().Equal(localNode.Address()) {
			continue
		}
		if _, found := m[n.Address()]; found {
			continue
		}
		m[n.Address()] = n
	}

	return &NodesState{localNode: localNode, nodes: m}
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
		if n.Address().Equal(ns.localNode.Address()) {
			return xerrors.Errorf("local node can be added")
		}

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
	lastBlock           *util.LockedItem
	lastINITVoteProof   *util.LockedItem
	lastACCEPTVoteProof *util.LockedItem
}

func NewLocalState(node *LocalNode, policy *LocalPolicy) *LocalState {
	// TODO fill this values from storage
	return &LocalState{
		node:                node,
		policy:              policy,
		nodes:               NewNodesState(node, nil),
		lastBlock:           util.NewLockedItem(nil),
		lastINITVoteProof:   util.NewLockedItem(nil),
		lastACCEPTVoteProof: util.NewLockedItem(nil),
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

func (ls *LocalState) LastBlock() Block {
	v := ls.lastBlock.Value()
	if v == nil {
		return nil
	}

	return v.(Block)
}

func (ls *LocalState) SetLastBlock(bk Block) *LocalState {
	_ = ls.lastBlock.SetValue(bk)

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
	v := ls.lastACCEPTVoteProof.Value()
	if v == nil {
		return nil
	}

	return v.(VoteProof)
}

func (ls *LocalState) SetLastACCEPTVoteProof(vp VoteProof) *LocalState {
	_ = ls.lastACCEPTVoteProof.SetValue(vp)

	return ls
}
