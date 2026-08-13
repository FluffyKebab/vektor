package nsw

import (
	"errors"
	"fmt"
	"sync"

	"github.com/FluffyKebab/vektor"
)

type NSW struct {
	dist          vektor.Distance
	graphObserver vektor.GraphObserver
	store         vektor.VectorStore

	queuePool     sync.Pool
	insertedPool  sync.Pool
	distancesPool sync.Pool

	entryNode *node
	nodes     map[uint32]*node

	maxNeighbors  int
	initNeighbors int
	efSearch      int
}

var _ vektor.Database = &NSW{}
var _ vektor.GraphMutater = &NSW{}

func New(store vektor.VectorStore, dist vektor.Distance, maxNeighbors, initNeighbors, efSearch int) *NSW {
	return &NSW{
		dist:          dist,
		store:         store,
		nodes:         make(map[uint32]*node),
		maxNeighbors:  maxNeighbors,
		initNeighbors: initNeighbors,
		efSearch:      efSearch,

		queuePool: sync.Pool{
			New: func() any {
				return newPriorityQueue(efSearch + 1)
			},
		},

		insertedPool: sync.Pool{
			New: func() any {
				return make(map[uint32]struct{}, efSearch)
			},
		},

		distancesPool: sync.Pool{
			New: func() any {
				return make(map[uint32]float64, efSearch)
			},
		},
	}
}

func (n *NSW) SetGraphObserver(obs vektor.GraphObserver) {
	n.graphObserver = obs
}

func (n *NSW) Insert(vec vektor.Vector) error {
	node := newNode(vec.ID(), n.maxNeighbors)

	if n.graphObserver != nil {
		n.graphObserver.OnNodeAdded(vec.ID())
	}

	n.store.Store(vec)
	n.nodes[vec.ID()] = node

	if n.entryNode == nil {
		n.entryNode = node
		return nil
	}

	s := doSearch(vec, n.initNeighbors, n)
	defer s.cleanup(n)

	if err := s.getErr(); err != nil {
		return fmt.Errorf("finding neighbors of new vector: %w", err)
	}

	for _, newNeighbor := range s.getResultNodes() {
		newNeighborVec, ok := n.store.Get(newNeighbor.id)
		if !ok {
			return errors.New("store is missing vector found in graph")
		}

		distNodeNeighbor, err := n.dist(vec, newNeighborVec)
		if err != nil {
			return fmt.Errorf("calculating vector distance: %w", err)
		}

		n.addNeighbor(node, newNeighbor, distNodeNeighbor)
		n.addNeighbor(newNeighbor, node, distNodeNeighbor)
	}

	return nil
}

func (n *NSW) FindClosest(vec vektor.Vector, k int) ([]vektor.Vector, error) {
	if n.entryNode == nil {
		if len(n.nodes) == 0 {
			return nil, errors.New("find closest with zero vectors inserted")
		}

		for _, entryNode := range n.nodes {
			n.entryNode = entryNode
			break
		}
	}

	s := doSearch(vec, k, n)
	result := s.getResultVectors()
	err := s.getErr()

	s.cleanup(n)
	return result, err
}

func (n *NSW) addNeighbor(n1, n2 *node, dist float64) {
	if n.graphObserver != nil {
		n.graphObserver.OnEdgeAdded(n1.id, n2.id, dist)
	}

	n1.neighbors = append(n1.neighbors, nodeDistance{n2, dist})

	if len(n1.neighbors) > n.maxNeighbors {
		n.removeWorstNeighbor(n1)
	}
}

func (n *NSW) removeWorstNeighbor(n1 *node) {
	if len(n1.neighbors) == 0 {
		return
	}

	worstIdx := n1.neighborWithHighestDistance()
	worstNeighborID := n1.neighbors[worstIdx].node.id

	if n.graphObserver != nil {
		n.graphObserver.OnEdgeRemoved(n1.id, worstNeighborID)
	}

	n1.removeNeighbor(worstIdx)
}

func (n *NSW) AddEdge(fromID uint32, toID uint32, dist float64) error {
	n1, ok := n.nodes[fromID]
	if !ok {
		return fmt.Errorf("adding edge: from_node %d does not exist in graph", fromID)
	}

	n2, ok := n.nodes[toID]
	if !ok {
		return fmt.Errorf("adding edge: to_node %d does not exist in graph", toID)
	}

	n1.neighbors = append(n1.neighbors, nodeDistance{n2, dist})
	return nil
}

func (n *NSW) AddNode(nodeID uint32) error {
	if _, ok := n.nodes[nodeID]; ok {
		return fmt.Errorf("adding node: %d already exist in graph", nodeID)
	}

	n.nodes[nodeID] = newNode(nodeID, n.maxNeighbors)

	if n.entryNode == nil {
		n.entryNode = n.nodes[nodeID]
	}

	return nil
}

func (n *NSW) RemoveEdge(fromID uint32, toID uint32) error {
	n1, ok := n.nodes[fromID]
	if !ok {
		return fmt.Errorf("removing edge: from_node %d does not exist in graph", fromID)
	}

	n2, ok := n.nodes[toID]
	if !ok {
		return fmt.Errorf("removing edge: to_node %d does not exist in graph", toID)
	}

	n2IdxInN1 := n1.findNeighborIndex(n2.id)
	if n2IdxInN1 == -1 {
		return fmt.Errorf("removing edge: from_id %d does not have neighbor %d", fromID, toID)
	}

	n1.removeNeighbor(n2IdxInN1)
	return nil
}
