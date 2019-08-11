package node

import "sort"

type SortBy func(n0, n1 Node) bool

func (sortBy SortBy) Sort(nodes []Node) {
	ns := &nodesSorter{
		nodes:  nodes,
		sortBy: sortBy,
	}
	sort.Sort(ns)
}

type nodesSorter struct {
	nodes  []Node
	sortBy func(n0, n1 Node) bool
}

func (s *nodesSorter) Len() int {
	return len(s.nodes)
}

// Swap is part of sort.Interface.
func (s *nodesSorter) Swap(i, j int) {
	s.nodes[i], s.nodes[j] = s.nodes[j], s.nodes[i]
}

func (s *nodesSorter) Less(i, j int) bool {
	return s.sortBy(s.nodes[i], s.nodes[j])
}

func SortByNodesByAddress(n0, n1 Node) bool {
	return n0.Address().String() < n1.Address().String()
}

func SortNodesByAddress(nodes []Node) {
	SortBy(SortByNodesByAddress).Sort(nodes)
}
