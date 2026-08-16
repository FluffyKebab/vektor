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
	maxIterations  int
}

var _ vektor.Database = &NSW{}
var _ vektor.GraphMutater = &NSW{}

func New(store vektor.VectorStore, dist vektor.Distance, maxNeighbors, initNeighbors, efSearch, efConstruction int) *NSW {
	n := &NSW{
		dist:           dist,
		vectors:        store,
		nodes:          newNodeStore(maxNeighbors + 1),
		maxNeighbors:   maxNeighbors,
		maxIterations:  300,
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
		return graph.NewVisitedSet(n.nodes.Len())
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
		n.entryNodeID = newNode.NodeID
		return nil, nil
	}

	result, err := n.doSearch(vec, n.initNeighbors, n.efConstruction)
	if err != nil {
		return nil, fmt.Errorf("finding neighbors of new vector: %w", err)
	}

	for _, nodeDist := range result {
		newNeighbor, found := n.nodes.Get(nodeDist.NodeID)
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
		if n.nodes.Len() == 0 {
			return nil, errors.New("find closest with zero vectors inserted")
		}
		entryNode, _ := n.nodes.Get(0)
		n.entryNodeID = entryNode.NodeID
	}

	result, err := n.doSearch(vec, k, n.efSearch)
	if err != nil {
		return nil, err
	}

	return n.getResultVectors(result)
}

func (n *NSW) doSearch(target vektor.Vector, k int, ef int) ([]graph.NodeDistance, error) {
	candidateQueue := n.candidateQueuePool.Get().(*graph.HeapQueue)
	resultQueue := n.resultQueuePool.Get().(*graph.SliceQueue)
	visited := n.visited.Get().(*graph.VisitedSet)
	defer n.cleanup(visited, candidateQueue, resultQueue)

	return graph.DoGreedySearch(
		target,
		n.entryNodeID,
		k,
		ef,
		n.maxIterations,
		candidateQueue,
		resultQueue,
		visited,
		n.dist,
		n.vectors,
		n.nodes,
	)
}

func (n *NSW) addNeighbor(n1, n2 graph.Node, dist float32) {
	if n.graphObserver != nil {
		n.graphObserver.OnEdgeAdded(n1.NodeID, n2.NodeID, dist)
	}

	needToRemove := n1.Neighbors.AddNeighbor(graph.NodeDistance{NodeID: n2.NodeID, Dist: dist})
	if needToRemove {
		n.removeWorstNeighbor(n1)
	}
}

func (n *NSW) removeWorstNeighbor(n1 graph.Node) {
	if len(n1.Neighbors) == 0 {
		return
	}

	worstIdx := n1.Neighbors.HighestDistanceIndex()
	worstNeighborID := n1.Neighbors[worstIdx].NodeID

	if n.graphObserver != nil {
		n.graphObserver.OnEdgeRemoved(n1.NodeID, worstNeighborID)
	}
	n1.Neighbors.Remove(worstIdx)
}

func (n *NSW) AddEdge(fromID uint32, toID uint32, dist float32) error {
	n1, ok := n.nodes.Get(fromID)
	if !ok {
		return fmt.Errorf("adding edge: from_node %d does not exist in graph", fromID)
	}

	_, ok = n.nodes.Get(toID)
	if !ok {
		return fmt.Errorf("adding edge: to_node %d does not exist in graph", toID)
	}

	n1.Neighbors.AddNeighbor(graph.NodeDistance{NodeID: toID, Dist: dist})
	return nil
}

func (n *NSW) AddNode(vectorID uint32) error {
	n.nodes.newNode(vectorID)
	return nil
}

func (n *NSW) RemoveEdge(fromID uint32, toID uint32) error {
	n1, ok := n.nodes.Get(fromID)
	if !ok {
		return fmt.Errorf("removing edge: from_node %d does not exist in graph", fromID)
	}

	n2, ok := n.nodes.Get(toID)
	if !ok {
		return fmt.Errorf("removing edge: to_node %d does not exist in graph", toID)
	}

	n2IdxInN1 := n1.Neighbors.IndexOf(n2.NodeID)
	if n2IdxInN1 == -1 {
		return fmt.Errorf("removing edge: from_id %d does not have neighbor %d", fromID, toID)
	}

	n1.Neighbors.Remove(n2IdxInN1)
	return nil
}

func (n *NSW) insertIntoVecAndNodeStore(v vektor.Vector) (vektor.Vector, graph.Node, error) {
	vecData, err := v.Read()
	if err != nil {
		return nil, graph.Node{}, fmt.Errorf("reading data from vector to be inserted: %w", err)
	}

	vecStoreID, err := n.vectors.Store(vecData)
	if err != nil {
		return nil, graph.Node{}, fmt.Errorf("storing error in vector store: %w", err)
	}

	newNode := n.nodes.newNode(vecStoreID)

	if n.graphObserver != nil {
		n.graphObserver.OnNodeAdded(vecStoreID)
	}

	return vektor.NewInMemoryVector(newNode.NodeID, vecData), newNode, nil
}

func (s *NSW) getResultVectors(nodeDist []graph.NodeDistance) ([]vektor.Vector, error) {
	result := make([]vektor.Vector, 0, len(nodeDist))

	for _, cur := range nodeDist {
		node, ok := s.nodes.Get(cur.NodeID)
		if !ok {
			return nil, errors.New("node in graph is missing from node store")
		}

		vector, ok := s.vectors.Get(node.VectorID)
		if !ok {
			return nil, errors.New("vector in graph is missing from vector store")
		}

		result = append(result, vektor.NewInMemoryVector(node.NodeID, vector))
	}

	return result, nil
}

func (n *NSW) cleanup(visited *graph.VisitedSet, candidates *graph.HeapQueue, results *graph.SliceQueue) {
	n.visited.Put(visited)

	candidates.Clear()
	n.candidateQueuePool.Put(candidates)

	results.Clear()
	n.resultQueuePool.Put(results)
}
