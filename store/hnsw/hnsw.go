package hnsw

import (
	"errors"
	"fmt"
	"math"
	"math/rand"

	"github.com/FluffyKebab/vektor"
	"github.com/FluffyKebab/vektor/store/graph"
)

type HNSW struct {
	nodes       *nodeStore
	vectors     vektor.VectorStore
	dist        vektor.Distance
	entryNodeID uint32

	maxIterations     int
	efUp              int
	efDown            int
	efConstruction    int
	initNeighborsUp   int
	initNeighborsDown int
}

var _ vektor.Database = &HNSW{}

func (n *HNSW) FindClosest(vec vektor.Vector, k int) ([]vektor.Vector, error) {
	if n.entryNodeID == graph.EmptyNodeID {
		if n.nodes.len() == 0 {
			return nil, errors.New("find closest with zero vectors inserted")
		}

		entryNode, err := n.nodes.get(0, uint8(n.nodes.layers-1))
		if err != nil {
			return nil, fmt.Errorf("node with ID 0 must be in top layer: %w", err)
		}

		n.entryNodeID = entryNode.nodeID
	}

	result, err := n.search(vec, n.entryNodeID, k, n.efUp, n.efDown, n.nodes.layers-1, 0)
	if err != nil {
		return nil, err
	}

	return n.getResultVectors(result)
}

func (n *HNSW) Insert(vec vektor.Vector) (vektor.Vector, error) {
	vecData, err := vec.Read()
	if err != nil {
		return nil, err
	}

	vectorID, err := n.vectors.Store(vecData)
	if err != nil {
		return nil, err
	}

	if n.nodes.len() == 0 {
		nodeID, err := n.nodes.add(vectorID, uint8(n.nodes.layers-1))
		if err != nil {
			return nil, err
		}
		n.entryNodeID = nodeID

		return vektor.NewInMemoryVector(nodeID, vecData), nil
	}

	layerToInsert := n.getLayerToInsert()
	newNodeID, err := n.nodes.add(vectorID, uint8(layerToInsert))
	if err != nil {
		return nil, err
	}

	result, err := n.search(vec, n.entryNodeID, 1, n.efUp, n.efDown, n.nodes.layers-1, layerToInsert+1)
	if err != nil {
		return nil, err
	}
	entryNode := result[0].NodeID

	for l := layerToInsert; l >= 0; l-- {
		newNode, err := n.nodes.get(newNodeID, uint8(l))
		if err != nil {
			return nil, err
		}

		numNeighbors := n.initNeighborsUp
		if l == 0 {
			numNeighbors = n.initNeighborsDown
		}

		neighbors, err := n.searchInLayer(vec, entryNode, numNeighbors, n.efConstruction, l)
		if err != nil {
			return nil, err
		}

		for _, newNeighbor := range neighbors {
			newNeighborNode, err := n.nodes.get(newNeighbor.NodeID, uint8(l))
			if err != nil {
				return nil, err
			}

			n.addNeighbor(uint8(l), newNode, newNeighbor.NodeID, newNeighbor.Dist)
			n.addNeighbor(uint8(l), newNeighborNode, newNodeID, newNeighbor.Dist)
		}
	}

	return vektor.NewInMemoryVector(newNodeID, vecData), nil
}

func (n *HNSW) getLayerToInsert() int {
	ml := 1 / math.Log(float64(n.nodes.maxN0))
	return int(math.Floor(-math.Log(rand.Float64()+0.00001) * ml))
}

func (n *HNSW) search(
	target vektor.Vector,
	entryNode uint32,
	k, efUp, efDown, startLayer, endLayer int,
) ([]graph.NodeDistance, error) {
	curNode := n.entryNodeID
	for l := startLayer; l > 0; l-- {
		result, err := n.searchInLayer(target, curNode, 1, efUp, l)
		if err != nil {
			return nil, err
		}

		curNode = result[0].NodeID

		if l <= endLayer {
			return result, nil
		}
	}

	return n.searchInLayer(target, curNode, k, efDown, 0)
}

func (n *HNSW) searchInLayer(target vektor.Vector, entry uint32, k int, ef int, layer int) ([]graph.NodeDistance, error) {
	nodes, err := n.nodes.getLayerNodeStore(uint8(layer))
	if err != nil {
		return nil, err
	}

	return graph.DoGreedySearch(
		target,
		entry,
		k,
		ef,
		n.maxIterations,
		graph.NewHeapQueue(200),
		graph.NewSliceQueue(200),
		graph.NewVisitedSet(200),
		n.dist,
		n.vectors,
		nodes,
	)
}

func (n *HNSW) addNeighbor(layer uint8, node node, id uint32, dist float32) {
	needToRemove := node.neighbors.AddNeighbor(graph.NodeDistance{NodeID: id, Dist: dist})
	if needToRemove {
		n.removeWorstNeighbor(node)
	}
}

func (n *HNSW) removeWorstNeighbor(n1 node) {
	if len(n1.neighbors) == 0 {
		return
	}

	worstIdx := n1.neighbors.HighestDistanceIndex()
	//worstNeighborID := n1.neighbors[worstIdx].NodeID
	n1.neighbors.Remove(worstIdx)
}

func (s *HNSW) getResultVectors(nodeDist []graph.NodeDistance) ([]vektor.Vector, error) {
	result := make([]vektor.Vector, 0, len(nodeDist))

	for _, cur := range nodeDist {
		node, err := s.nodes.get(cur.NodeID, 0)
		if err != nil {
			return nil, fmt.Errorf("node in graph is missing from node store: %w", err)
		}

		vector, ok := s.vectors.Get(node.vectorStoreID)
		if !ok {
			return nil, errors.New("vector in graph is missing from vector store")
		}

		result = append(result, vektor.NewInMemoryVector(node.nodeID, vector))
	}

	return result, nil
}
