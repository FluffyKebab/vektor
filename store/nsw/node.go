package nsw

import "github.com/FluffyKebab/vektor/store/graph"

type nodeStore struct {
	neighbors []graph.NodeDistance
	vectorIDs []uint32
	maxN      int
}

var _ graph.NodeStore = &nodeStore{}

func newNodeStore(maxN int) *nodeStore {
	return &nodeStore{
		neighbors: make([]graph.NodeDistance, 0),
		vectorIDs: make([]uint32, 0),
		maxN:      maxN,
	}
}

func (s *nodeStore) Get(id uint32) (graph.Node, error) {
	start := int(id) * s.maxN
	end := start + s.maxN

	if end > len(s.neighbors) || int(id) >= len(s.vectorIDs) {
		return graph.Node{}, graph.ErrInvalidID
	}

	return graph.Node{
		Neighbors: s.neighbors[start:end],
		NodeID:    id,
		VectorID:  s.vectorIDs[int(id)],
	}, nil
}

func (s *nodeStore) newNode(vectorStoreID uint32) graph.Node {
	id := len(s.vectorIDs)

	prevLength := len(s.neighbors)
	s.neighbors = append(s.neighbors, make([]graph.NodeDistance, s.maxN)...)
	for i := prevLength; i < len(s.neighbors); i++ {
		s.neighbors[i].NodeID = graph.EmptyNodeID
	}
	s.vectorIDs = append(s.vectorIDs, vectorStoreID)

	res, _ := s.Get(uint32(id))
	return res
}

func (s *nodeStore) Len() int {
	return len(s.vectorIDs)
}
