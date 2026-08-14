package nsw

import (
	"math"
)

const emptyNodeID uint32 = math.MaxUint32

type node struct {
	nodeStoreID   uint32
	vectorStoreID uint32
	neighbors     []nodeDistance
}

type nodeDistance struct {
	nodeID uint32
	dist   float32
}

func (n node) indexOfNeighborWithHighestDistance() int {
	if n.numNeighbors() == 0 {
		return -1
	}

	worstIdx := 0
	maxDist := n.neighbors[0].dist
	for i := 1; i < len(n.neighbors) && n.neighbors[i].nodeID != emptyNodeID; i++ {
		if n.neighbors[i].dist > maxDist {
			maxDist = n.neighbors[i].dist
			worstIdx = i
		}
	}

	return worstIdx
}

func (n node) indexOfNeighbor(neighborID uint32) int {
	for i := 0; i < len(n.neighbors) && n.neighbors[i].nodeID != emptyNodeID; i++ {
		if n.neighbors[i].nodeID == neighborID {
			return i
		}
	}

	return -1
}

func (n node) numNeighbors() int {
	i := 0
	for i < len(n.neighbors) && n.neighbors[i].nodeID != emptyNodeID {
		i++
	}
	return i
}

func (n *node) removeNeighbor(i int) {
	lastIdx := n.numNeighbors() - 1
	n.neighbors[i] = n.neighbors[lastIdx]
	n.neighbors[lastIdx] = nodeDistance{nodeID: emptyNodeID}
}

func (n *node) addNeighbor(nd nodeDistance) (needToRemove bool) {
	place := n.numNeighbors()
	if place >= len(n.neighbors) {
		return true
	}

	n.neighbors[place] = nd
	if place+1 >= len(n.neighbors) {
		return true
	}

	return false
}

type nodeStore struct {
	neighbors []nodeDistance
	vectorIDs []uint32
	nextID    uint32
	maxN      int
}

func newNodeStore(maxN int) *nodeStore {
	return &nodeStore{
		neighbors: make([]nodeDistance, 0),
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

	return node{id, s.vectorIDs[int(id)], s.neighbors[start:end]}, true
}

func (s *nodeStore) getNeighbors(id uint32) []nodeDistance {
	start := int(id) * s.maxN
	end := start + s.maxN
	return s.neighbors[start:end]
}

func (s *nodeStore) newNode(vectorStoreID uint32) node {
	id := s.nextID
	s.nextID++

	newBlock := make([]nodeDistance, s.maxN)
	for i := range newBlock {
		newBlock[i].nodeID = emptyNodeID
	}
	s.neighbors = append(s.neighbors, newBlock...)
	s.vectorIDs = append(s.vectorIDs, vectorStoreID)

	newNode, _ := s.get(id)
	return newNode
}

func (s *nodeStore) numNodes() int {
	return int(s.nextID)
}
