package hnsw

import (
	"errors"
	"fmt"

	"github.com/FluffyKebab/vektor/store/graph"
)

type node struct {
	neighbors     graph.Neighbors
	nodeID        uint32
	vectorStoreID uint32
	highestLevel  uint8
}

type nodeStore struct {
	layerLengths []int

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
		layerLengths:         make([]int, layers),
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

func (n *nodeStore) get(nodeID uint32, layer uint8) (node, error) {
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
		start := int(baseOffset) + int(layer-1)*n.maxN1
		end := start + n.maxN1
		ns = n.upperLayerNeighbors[start:end]
	}

	return node{
		neighbors:     ns,
		vectorStoreID: vecID,
		nodeID:        nodeID,
		highestLevel:  layer,
	}, nil
}

func (n *nodeStore) add(vectorStoreID uint32, highestLevel uint8) (uint32, error) {
	if highestLevel >= uint8(n.layers) {
		return 0, fmt.Errorf("invalid layer %d", highestLevel)
	}

	nodeID := uint32(len(n.vectorStoreIDs))
	n.vectorStoreIDs = append(n.vectorStoreIDs, vectorStoreID)
	n.highestLevel = append(n.highestLevel, highestLevel)

	n.allocateBottomLayer()
	n.allocateUpperLayers(int(highestLevel))

	return nodeID, nil
}

func (n *nodeStore) allocateBottomLayer() {
	start := len(n.bottomLayerNeighbors)

	n.bottomLayerNeighbors = append(n.bottomLayerNeighbors, make([]graph.NodeDistance, n.maxN0)...)
	for i := start; i < start+n.maxN0; i++ {
		n.bottomLayerNeighbors[i].NodeID = graph.EmptyNodeID
	}

	n.layerLengths[0]++
}

func (n *nodeStore) allocateUpperLayers(highestLevel int) {
	offset := len(n.upperLayerNeighbors)
	n.offsets = append(n.offsets, uint32(offset))
	if highestLevel == 0 {
		return
	}

	length := int(highestLevel) * n.maxN1

	n.upperLayerNeighbors = append(n.upperLayerNeighbors, make([]graph.NodeDistance, length)...)
	for i := offset; i < offset+length; i++ {
		n.upperLayerNeighbors[i].NodeID = graph.EmptyNodeID
	}

	for i := 1; i <= highestLevel; i++ {
		n.layerLengths[i]++
	}
}

func (n *nodeStore) len() int {
	return len(n.vectorStoreIDs)
}

func (n *nodeStore) getLayerNodeStore(layer uint8) (layerNodeStore, error) {
	if layer >= uint8(n.layers) {
		return layerNodeStore{}, errors.New("invalid layer")
	}

	return layerNodeStore{n, layer}, nil
}

type layerNodeStore struct {
	n     *nodeStore
	layer uint8
}

var _ graph.NodeStore = layerNodeStore{}

func (n layerNodeStore) Get(id uint32) (graph.Node, error) {
	result, err := n.n.get(id, n.layer)

	return graph.Node{
		Neighbors: result.neighbors,
		NodeID:    result.nodeID,
		VectorID:  result.vectorStoreID,
	}, err
}

func (n layerNodeStore) Len() int {
	return n.n.layerLengths[n.layer]
}
