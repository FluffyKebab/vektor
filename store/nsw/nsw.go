package nsw

import (
	"errors"
	"fmt"

	"github.com/FluffyKebab/vektor"
)

type node struct {
	id        uint32
	neighbors []nodeDistance
}

type nodeDistance struct {
	node *node
	dist float64
}

func newNode(v vektor.Vector, maxN int) *node {
	return &node{id: v.ID(), neighbors: make([]nodeDistance, 0, maxN+1)}
}

type NSW struct {
	dist          vektor.Distance
	graphObserver vektor.GraphObserver
	store         vektor.VectorStore
	entryNode     *node
	maxNeighbors  int
	initNeighbors int
}

var _ vektor.Database = &NSW{}

func NewNSW(store vektor.VectorStore, dist vektor.Distance, obs vektor.GraphObserver) *NSW {
	return &NSW{
		dist:          dist,
		graphObserver: obs,
		store:         store,
	}
}

func (n *NSW) Insert(vec vektor.Vector) error {
	node := newNode(vec, n.maxNeighbors)

	if n.graphObserver != nil {
		n.graphObserver.OnNodeAdded(vec.ID())
	}

	n.store.Store(vec)

	if n.entryNode == nil {
		n.entryNode = node
		return nil
	}

	s := doSearch(vec, n.initNeighbors, n.dist, n.entryNode, n.store)
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
	s := doSearch(vec, k, n.dist, n.entryNode, n.store)
	return s.getResultVectors(), s.getErr()
}

func (n *NSW) addNeighbor(n1, n2 *node, dist float64) {
	if n.graphObserver != nil {
		n.graphObserver.OnEdgeAdded(n1.id, n2.id)
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

	worstIdx := 0
	maxDist := n1.neighbors[0].dist
	for i := 1; i < len(n1.neighbors); i++ {
		if n1.neighbors[i].dist > maxDist {
			maxDist = n1.neighbors[i].dist
			worstIdx = i
		}
	}
	worstNeighborID := n1.neighbors[worstIdx].node.id

	if n.graphObserver != nil {
		n.graphObserver.OnEdgeRemoved(n1.id, worstNeighborID)
	}

	lastIdx := len(n1.neighbors) - 1
	n1.neighbors[worstIdx] = n1.neighbors[lastIdx]
	n1.neighbors[lastIdx] = nodeDistance{}
	n1.neighbors = n1.neighbors[:lastIdx]
}

type searcher struct {
	targetVec       vektor.Vector
	candidateQueue  *priorityQueue
	resultQueue     *priorityQueue
	insertedInQueue map[uint32]struct{}
	dist            vektor.Distance
	store           vektor.VectorStore
	distances       map[uint32]float64
	k               int
	err             error
}

func newSearcher(targetVec vektor.Vector, k int, dist vektor.Distance, entryNode *node, store vektor.VectorStore) *searcher {
	s := &searcher{
		targetVec:       targetVec,
		candidateQueue:  newPriorityQueue(),
		resultQueue:     newPriorityQueue(),
		dist:            dist,
		distances:       make(map[uint32]float64),
		insertedInQueue: make(map[uint32]struct{}),
		store:           store,
		k:               k,
	}

	entryVec, ok := s.store.Get(entryNode.id)
	if !ok {
		s.err = errors.New("entry node vector is missing from vectore store")
		return s
	}

	entryNodeDist, err := s.getDist(entryVec, targetVec)
	if err != nil {
		s.err = err
	}

	s.insertInQueue(entryNode, entryNodeDist)
	return s
}

func doSearch(vec vektor.Vector, k int, dist vektor.Distance, entryNode *node, store vektor.VectorStore) *searcher {
	s := newSearcher(vec, k, dist, entryNode, store)
	for s.next() {
	}
	return s
}

func (s *searcher) next() bool {
	if s.err != nil {
		return false
	}

	if s.candidateQueue.len() == 0 {
		return false
	}

	curBestCandidate, err := s.candidateQueue.popTop()
	if err != nil {
		s.err = err
		return false
	}

	var curWorstResult nodeDistance
	if s.resultQueue.len() != 0 {
		curWorstResult, err = s.resultQueue.getBottom()
		if err != nil {
			s.err = err
			return false
		}

		if curWorstResult.dist < curBestCandidate.dist {
			return false
		}
	}

	for _, n := range curBestCandidate.node.neighbors {

		s.handleNeighbor(n.node, curWorstResult)
	}

	return true
}

func (s *searcher) handleNeighbor(neighbor *node, curWorstResult nodeDistance) {
	nVec, ok := s.store.Get(neighbor.id)
	if !ok {
		s.err = errors.New("vector in graph is missing from vectore store")
		return
	}

	dist, err := s.getDist(nVec, s.targetVec)
	if err != nil {
		s.err = err
		return
	}

	if s.resultQueue.len() <= s.k ||
		(curWorstResult.node != nil && dist < curWorstResult.dist) {

		s.insertInQueue(neighbor, dist)
	}
}

func (s *searcher) insertInQueue(n *node, dist float64) {
	if _, isInserted := s.insertedInQueue[n.id]; isInserted {
		return
	}
	s.insertedInQueue[n.id] = struct{}{}

	elem := nodeDistance{n, dist}

	s.resultQueue.insert(elem)
	if s.resultQueue.len() > s.k {
		s.resultQueue.removeBottom()
	}

	s.candidateQueue.insert(elem)
}

func (s *searcher) getDist(curVec, targetVec vektor.Vector) (float64, error) {
	if d, found := s.distances[curVec.ID()]; found {
		return d, nil
	}

	d, err := s.dist(curVec, targetVec)
	if err != nil {
		return 0, err
	}

	s.distances[curVec.ID()] = d
	return d, nil
}

func (s *searcher) getErr() error {
	if s.resultQueue.len() == 0 {
		if s.err != nil {
			s.err = fmt.Errorf("no results found: %w", s.err)
		} else {
			s.err = errors.New("no results found")
		}
	}

	return s.err
}

func (s *searcher) getResultVectors() []vektor.Vector {
	result := make([]vektor.Vector, s.resultQueue.len())
	for i, vd := range s.resultQueue.getSlice() {
		vec, ok := s.store.Get(vd.id)
		if !ok {
			s.err = errors.New("vector in graph is missing from vector store")
			return nil
		}

		result[i] = vec
	}

	return result
}

func (s *searcher) getResultNodes() []*node {
	return s.resultQueue.getSlice()
}
