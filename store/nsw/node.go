package nsw

import "github.com/FluffyKebab/vektor/store/graph"

type node struct {
	neighbors graph.Neighbors
	nodeID    uint32
	vectorID  uint32
}

type nodeStore struct {
	neighbors []graph.NodeDistance
	vectorIDs []uint32
	nextID    uint32
	maxN      int
}

func newNodeStore(maxN int) *nodeStore {
	return &nodeStore{
		neighbors: make([]graph.NodeDistance, 0),
		vectorIDs: make([]uint32, 0),
		maxN:      maxN,
	}
}

func (s *nodeStore) get(id uint32) (node, bool) {
	start := int(id) * s.maxN
	end := start + s.maxN

	if end > len(s.neighbors) || int(id) >= len(s.vectorIDs) {
		return node{}, false
	}

	return node{s.neighbors[start:end], id, s.vectorIDs[int(id)]}, true
}

func (s *nodeStore) newNode(vectorStoreID uint32) node {
	id := s.nextID
	s.nextID++

	prevLength := len(s.neighbors)
	s.neighbors = append(s.neighbors, make([]graph.NodeDistance, s.maxN)...)
	for i := prevLength; i < len(s.neighbors); i++ {
		s.neighbors[i].NodeID = graph.EmptyNodeID
	}
	s.vectorIDs = append(s.vectorIDs, vectorStoreID)

	newNode, _ := s.get(id)
	return newNode
}

func (s *nodeStore) numNodes() int {
	return int(s.nextID)
}
