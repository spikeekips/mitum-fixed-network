package main

import (
	"sort"
	"sync"
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

	var nodes []*Node
	for i, name := range nodeNames[:config.NumberOfNodes()] {
		no, err := NewNode(uint(i), config, config.Nodes[name])
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
		go func() {
			errChan <- n.Start()
			wg.Done()
		}()
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
		go func() {
			errChan <- n.Stop()
			wg.Done()
		}()
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
