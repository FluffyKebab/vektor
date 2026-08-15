package hnsw

import (
	"errors"
	"fmt"
	"math"
)

var errInvalidID = errors.New("invalid node ID")

const emptyNodeID uint32 = math.MaxUint32

type nodeDistance struct {
	nodeID uint32
	dist   float32
}

type neighbors []nodeDistance

func (n neighbors) indexOfNeighborWithHighestDistance() int {
	if n.numNeighbors() == 0 {
		return -1
	}

	worstIdx := 0
	maxDist := n[0].dist
	for i := 1; i < len(n) && n[i].nodeID != emptyNodeID; i++ {
		if n[i].dist > maxDist {
			maxDist = n[i].dist
			worstIdx = i
		}
	}

	return worstIdx
}

func (n neighbors) indexOfNeighbor(neighborID uint32) int {
	for i := 0; i < len(n) && n[i].nodeID != emptyNodeID; i++ {
		if n[i].nodeID == neighborID {
			return i
		}
	}

	return -1
}

func (n neighbors) numNeighbors() int {
	i := 0
	for i < len(n) && n[i].nodeID != emptyNodeID {
		i++
	}
	return i
}

func (n neighbors) removeNeighbor(i int) {
	lastIdx := n.numNeighbors() - 1
	n[i] = n[lastIdx]
	n[lastIdx] = nodeDistance{nodeID: emptyNodeID}
}

func (n neighbors) addNeighbor(nd nodeDistance) (needToRemove bool) {
	place := n.numNeighbors()
	if place >= len(n) {
		return true
	}

	n[place] = nd
	if place+1 >= len(n) {
		return true
	}

	return false
}

type node struct {
	neighbors     neighbors
	vectorStoreID uint32
	highestLevel  uint8
}

type nodeStore struct {
	bottomLayerNeighbors []nodeDistance
	vectorStoreIDs       []uint32
	highestLevel         []uint8

	// offsets functions a map from a nodeID to the index where storage of upper layer neighbors begin.
	offsets             []uint32
	upperLayerNeighbors []nodeDistance

	// layers stores the number of layers.
	layers int

	// maxN0 stores the maximum numbers of neighbors a node can have in the bottom layer.
	maxN0 int

	// maxN1 stores the maximum numbers of neighbors a node can have in upper layers.
	maxN1 int
}

func newNodeStore(layers, maxN0, maxN1, initCap int) *nodeStore {
	return &nodeStore{
		bottomLayerNeighbors: make([]nodeDistance, 0, initCap),
		vectorStoreIDs:       make([]uint32, 0, initCap),
		highestLevel:         make([]uint8, 0, initCap),
		offsets:              make([]uint32, 0, initCap),
		upperLayerNeighbors:  make([]nodeDistance, 0),

		layers: layers,
		maxN0:  maxN0,
		maxN1:  maxN1,
	}
}

func (n *nodeStore) getNode(nodeID uint32, layer uint8) (node, error) {
	if int(nodeID) >= len(n.highestLevel) {
		return node{}, errInvalidID
	}

	if layer > n.highestLevel[nodeID] {
		return node{}, fmt.Errorf("node %d does not exist in layer %d", nodeID, layer)
	}

	vecID := n.vectorStoreIDs[nodeID]

	var ns neighbors
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
		n.upperLayerNeighbors = append(n.upperLayerNeighbors, make([]nodeDistance, length)...)
		for i := offset; i < offset+length; i++ {
			n.bottomLayerNeighbors[i].nodeID = emptyNodeID
		}
	}

	return nodeID, nil
}
