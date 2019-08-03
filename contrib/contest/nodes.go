package main

import (
	"sort"
	"sync"

	"github.com/spikeekips/mitum/node"
)

type Nodes struct {
	sync.RWMutex
	nodes []*Node
}

func NewNodes(config *Config) (*Nodes, error) {
	// create node
	var nodeNames []string
	for n := range config.Nodes {
		nodeNames = append(nodeNames, n)
	}
	sort.Strings(nodeNames)

	var nodeList []node.Node
	for i, name := range nodeNames[:config.NumberOfNodes()] {
		n := NewHome(uint(i)).SetAlias(name)
		nodeList = append(nodeList, n)
	}

	var nodes []*Node
	for _, n := range nodeList {
		nodeConfig := config.Nodes[n.Alias()]

		no, err := NewNode(
			n.(node.Home),
			nodeList,
			config,
			nodeConfig,
		)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, no)
	}

	// connect network
	for _, n := range nodes {
		for _, o := range nodes {
			n.nt.AddMembers(o.nt)
		}
	}

	ns := &Nodes{
		nodes: nodes,
	}

	return ns, nil
}

func (ns *Nodes) Start() error {
	ns.Lock()
	defer ns.Unlock()

	var wg sync.WaitGroup
	wg.Add(len(ns.nodes))

	errChan := make(chan error, len(ns.nodes))

	for _, n := range ns.nodes {
		go func(n *Node) {
			errChan <- n.Start()
			wg.Done()
		}(n)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func (ns *Nodes) Stop() error {
	ns.Lock()
	defer ns.Unlock()

	var wg sync.WaitGroup
	wg.Add(len(ns.nodes))

	errChan := make(chan error, len(ns.nodes))

	for _, n := range ns.nodes {
		go func(n *Node) {
			errChan <- n.Stop()
			wg.Done()
		}(n)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}
