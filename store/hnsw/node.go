package hnsw

import (
	"fmt"

	"github.com/FluffyKebab/vektor/store/graph"
)

type node struct {
	neighbors     graph.Neighbors
	vectorStoreID uint32
	highestLevel  uint8
}

type nodeStore struct {
	bottomLayerNeighbors []graph.NodeDistance
	vectorStoreIDs       []uint32
	highestLevel         []uint8

	// offsets functions a map from a nodeID to the index where storage of upper layer neighbors begin.
	offsets             []uint32
	upperLayerNeighbors []graph.NodeDistance

	// layers stores the number of layers.
	layers int

	// maxN0 stores the maximum numbers of neighbors a node can have in the bottom layer.
	maxN0 int

	// maxN1 stores the maximum numbers of neighbors a node can have in upper layers.
	maxN1 int
}

func newNodeStore(layers, maxN0, maxN1, initCap int) *nodeStore {
	return &nodeStore{
		bottomLayerNeighbors: make([]graph.NodeDistance, 0, initCap),
		vectorStoreIDs:       make([]uint32, 0, initCap),
		highestLevel:         make([]uint8, 0, initCap),
		offsets:              make([]uint32, 0, initCap),
		upperLayerNeighbors:  make([]graph.NodeDistance, 0),

		layers: layers,
		maxN0:  maxN0,
		maxN1:  maxN1,
	}
}

func (n *nodeStore) getNode(nodeID uint32, layer uint8) (node, error) {
	if int(nodeID) >= len(n.highestLevel) {
		return node{}, graph.ErrInvalidID
	}

	if layer > n.highestLevel[nodeID] {
		return node{}, fmt.Errorf("node %d does not exist in layer %d", nodeID, layer)
	}

	vecID := n.vectorStoreIDs[nodeID]

	var ns graph.Neighbors
	if layer == 0 {
		start := int(nodeID) * n.maxN0
		end := start + n.maxN0
		ns = n.bottomLayerNeighbors[start:end]
	} else {
		baseOffset := n.offsets[nodeID]
		start := int(baseOffset) + int(layer)*n.maxN1
		end := start + n.maxN1
		ns = n.upperLayerNeighbors[start:end]
	}

	return node{
		neighbors:     ns,
		vectorStoreID: vecID,
		highestLevel:  layer,
	}, nil
}

func (n *nodeStore) addNode(vectorStoreID uint32, highestLevel uint8) (uint32, error) {
	if highestLevel >= uint8(n.layers) {
		return 0, fmt.Errorf("invalid layer %d, node store only has %d layers", highestLevel, n.layers)
	}

	nodeID := uint32(len(n.vectorStoreIDs))
	n.vectorStoreIDs = append(n.vectorStoreIDs, vectorStoreID)
	n.highestLevel = append(n.highestLevel, highestLevel)
	n.offsets = append(n.offsets, 0)

	if highestLevel != 0 {
		offset := len(n.upperLayerNeighbors)
		length := (n.layers - 1) * n.maxN0

		n.offsets[nodeID] = uint32(offset)
		n.upperLayerNeighbors = append(n.upperLayerNeighbors, make([]graph.NodeDistance, length)...)
		for i := offset; i < offset+length; i++ {
			n.bottomLayerNeighbors[i].NodeID = graph.EmptyNodeID
		}
	}

	return nodeID, nil
}
