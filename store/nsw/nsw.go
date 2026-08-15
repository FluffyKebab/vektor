package nsw

import (
	"errors"
	"fmt"
	"sync"

	"github.com/FluffyKebab/vektor"
	"github.com/FluffyKebab/vektor/store/graph"
)

type NSW struct {
	dist          vektor.Distance
	graphObserver vektor.GraphObserver
	vectors       vektor.VectorStore

	resultQueuePool    sync.Pool
	candidateQueuePool sync.Pool
	visited            sync.Pool

	entryNodeID uint32
	nodes       *nodeStore

	maxNeighbors   int
	initNeighbors  int
	efSearch       int
	efConstruction int
	curID          uint32
}

var _ vektor.Database = &NSW{}
var _ vektor.GraphMutater = &NSW{}

func New(store vektor.VectorStore, dist vektor.Distance, maxNeighbors, initNeighbors, efSearch, efConstruction int) *NSW {
	n := &NSW{
		dist:           dist,
		vectors:        store,
		nodes:          newNodeStore(maxNeighbors + 1),
		maxNeighbors:   maxNeighbors,
		initNeighbors:  initNeighbors,
		efSearch:       efSearch,
		efConstruction: efConstruction,
		entryNodeID:    graph.EmptyNodeID,

		resultQueuePool: sync.Pool{
			New: func() any {
				return graph.NewSliceQueue(efSearch + 1)
			},
		},

		candidateQueuePool: sync.Pool{
			New: func() any {
				return graph.NewHeapQueue(efSearch * 2)
			},
		},
	}

	n.visited.New = func() any {
		return graph.NewVisitedSet(n.nodes.numNodes())
	}

	return n
}

func (n *NSW) SetGraphObserver(obs vektor.GraphObserver) {
	n.graphObserver = obs
}

func (n *NSW) Insert(vec vektor.Vector) (vektor.Vector, error) {
	resultVec, newNode, err := n.insertIntoVecAndNodeStore(vec)
	if err != nil {
		return nil, err
	}

	if n.entryNodeID == graph.EmptyNodeID {
		n.entryNodeID = newNode.nodeID
		return nil, nil
	}

	s := doSearch(vec, n.initNeighbors, n.efConstruction, n)
	defer s.cleanup(n)
	if err := s.getErr(); err != nil {
		return nil, fmt.Errorf("finding neighbors of new vector: %w", err)
	}

	for _, nodeDist := range s.getResultNodeDist() {
		newNeighbor, found := n.nodes.get(nodeDist.NodeID)
		if !found {
			return nil, errors.New("node unexpectedly missing from node store")
		}

		n.addNeighbor(newNode, newNeighbor, nodeDist.Dist)
		n.addNeighbor(newNeighbor, newNode, nodeDist.Dist)
	}

	return resultVec, nil
}

func (n *NSW) FindClosest(vec vektor.Vector, k int) ([]vektor.Vector, error) {
	if n.entryNodeID == graph.EmptyNodeID {
		if n.nodes.numNodes() == 0 {
			return nil, errors.New("find closest with zero vectors inserted")
		}
		entryNode, _ := n.nodes.get(0)
		n.entryNodeID = entryNode.nodeID
	}

	s := doSearch(vec, k, n.efSearch, n)
	result := s.getResultVectors()
	err := s.getErr()
	s.cleanup(n)

	return result, err
}

func (n *NSW) addNeighbor(n1, n2 node, dist float32) {
	if n.graphObserver != nil {
		n.graphObserver.OnEdgeAdded(n1.nodeID, n2.nodeID, dist)
	}

	needToRemove := n1.neighbors.AddNeighbor(graph.NodeDistance{NodeID: n2.nodeID, Dist: dist})
	if needToRemove {
		n.removeWorstNeighbor(n1)
	}
}

func (n *NSW) removeWorstNeighbor(n1 node) {
	if len(n1.neighbors) == 0 {
		return
	}

	worstIdx := n1.neighbors.HighestDistanceIndex()
	worstNeighborID := n1.neighbors[worstIdx].NodeID

	if n.graphObserver != nil {
		n.graphObserver.OnEdgeRemoved(n1.nodeID, worstNeighborID)
	}
	n1.neighbors.Remove(worstIdx)
}

func (n *NSW) AddEdge(fromID uint32, toID uint32, dist float32) error {
	n1, ok := n.nodes.get(fromID)
	if !ok {
		return fmt.Errorf("adding edge: from_node %d does not exist in graph", fromID)
	}

	_, ok = n.nodes.get(toID)
	if !ok {
		return fmt.Errorf("adding edge: to_node %d does not exist in graph", toID)
	}

	n1.neighbors.AddNeighbor(graph.NodeDistance{NodeID: toID, Dist: dist})
	return nil
}

func (n *NSW) AddNode(vectorID uint32) error {
	n.nodes.newNode(vectorID)
	return nil
}

func (n *NSW) RemoveEdge(fromID uint32, toID uint32) error {
	n1, ok := n.nodes.get(fromID)
	if !ok {
		return fmt.Errorf("removing edge: from_node %d does not exist in graph", fromID)
	}

	n2, ok := n.nodes.get(toID)
	if !ok {
		return fmt.Errorf("removing edge: to_node %d does not exist in graph", toID)
	}

	n2IdxInN1 := n1.neighbors.IndexOf(n2.nodeID)
	if n2IdxInN1 == -1 {
		return fmt.Errorf("removing edge: from_id %d does not have neighbor %d", fromID, toID)
	}

	n1.neighbors.Remove(n2IdxInN1)
	return nil
}

func (n *NSW) insertIntoVecAndNodeStore(v vektor.Vector) (vektor.Vector, node, error) {
	vecData, err := v.Read()
	if err != nil {
		return nil, node{}, fmt.Errorf("reading data from vector to be inserted: %w", err)
	}

	vecStoreID, err := n.vectors.Store(vecData)
	if err != nil {
		return nil, node{}, fmt.Errorf("storing error in vector store: %w", err)
	}

	newNode := n.nodes.newNode(vecStoreID)

	if n.graphObserver != nil {
		n.graphObserver.OnNodeAdded(vecStoreID)
	}

	return vektor.NewInMemoryVector(newNode.nodeID, vecData), newNode, nil
}
