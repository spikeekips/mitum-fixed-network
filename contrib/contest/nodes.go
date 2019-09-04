package main

import (
	"sync"

	"github.com/spikeekips/mitum/node"
)

type Nodes struct {
	sync.RWMutex
	nodes []*Node
}

func NewNodes(config *Config, nodeList []node.Node) (*Nodes, error) { // nolint
	var wg sync.WaitGroup
	wg.Add(len(nodeList))

	nch := make(chan *Node)
	for _, n := range nodeList {
		nodeConfig := config.Nodes[n.Alias()]

		go func(n node.Node, c *NodeConfig) {
			no, err := NewNode(
				interface{}(n).(node.Home),
				nodeList,
				config,
				c,
			)
			if err != nil {
				panic(err)
			}

			nch <- no
		}(n, nodeConfig)
	}

	var nodes []*Node
	for no := range nch {
		nodes = append(nodes, no)
		wg.Done()
		if len(nodes) == len(nodeList) {
			break
		}
	}
	close(nch)

	wg.Wait()

	// connect network
	for _, n := range nodes {
		for _, o := range nodes {
			n.nt.AddMembers(o.nt)
		}
	}

	return &Nodes{nodes: nodes}, nil
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
