package nsw

import (
	"errors"
	"fmt"

	"github.com/FluffyKebab/vektor"
)

type node struct {
	v         vektor.Vector
	neighbors []nodeDistance
}

type nodeDistance struct {
	node *node
	dist float64
}

func newNode(v vektor.Vector, maxN int) *node {
	return &node{v: v, neighbors: make([]nodeDistance, 0, maxN+1)}
}

type NSW struct {
	entryNode     *node
	dist          vektor.Distance
	graphObserver vektor.GraphObserver
	maxNeighbors  int
	initNeighbors int
}

var _ vektor.Database = &NSW{}

func (n *NSW) Insert(vec vektor.Vector) error {
	node := newNode(vec, n.maxNeighbors)
	n.graphObserver.OnNodeAdded(vec.ID())

	if n.entryNode == nil {
		n.entryNode = node
		return nil
	}

	s := doSearch(vec, n.initNeighbors, n.dist, n.entryNode)
	if err := s.getErr(); err != nil {
		return fmt.Errorf("finding neighbors of new vector: %w", err)
	}

	for _, newNeighbor := range s.getResultNodes() {
		distNodeNeighbor, err := n.dist(vec, newNeighbor.v)
		if err != nil {
			return fmt.Errorf("calculating vector distance: %w", err)
		}

		node.neighbors = append(node.neighbors, nodeDistance{newNeighbor, distNodeNeighbor})
		newNeighbor.neighbors = append(newNeighbor.neighbors, nodeDistance{node, distNodeNeighbor})

		if len(newNeighbor.neighbors) > n.maxNeighbors {
			n.removeWorstNeighbor(newNeighbor)
		}
	}

	return nil
}

func (n *NSW) FindClosest(vec vektor.Vector, k int) ([]vektor.Vector, error) {
	s := doSearch(vec, k, n.dist, n.entryNode)
	return s.getResultVectors(), s.getErr()
}

func (n *NSW) addNeighbor(n1, n2 *node, dist float64) {
	n.graphObserver.OnEdgeAdded(n1.v.ID(), n2.v.ID())
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
	worstNeighborID := n1.neighbors[worstIdx].node.v.ID()

	n.graphObserver.OnEdgeRemoved(n1.v.ID(), worstNeighborID)

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
	distances       map[uint32]float64
	k               int
	err             error
}

func doSearch(vec vektor.Vector, k int, dist vektor.Distance, entryNode *node) *searcher {
	s := newSearcher(vec, k, dist, entryNode)
	for s.next() {
	}
	return s
}

func newSearcher(targetVec vektor.Vector, k int, dist vektor.Distance, entryNode *node) *searcher {
	s := &searcher{
		targetVec:       targetVec,
		candidateQueue:  newPriorityQueue(),
		resultQueue:     newPriorityQueue(),
		dist:            dist,
		distances:       make(map[uint32]float64),
		insertedInQueue: make(map[uint32]struct{}),
		k:               k,
	}

	entryNodeDist, err := s.getDist(entryNode.v, targetVec)
	if err != nil {
		s.err = err
	}

	s.insertInQueue(entryNode, entryNodeDist)
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
	dist, err := s.getDist(neighbor.v, s.targetVec)
	if err != nil {
		s.err = err
		return
	}

	if s.resultQueue.len() < s.k ||
		(curWorstResult.node != nil && dist < curWorstResult.dist) {

		s.insertInQueue(neighbor, dist)
	}
}

func (s *searcher) insertInQueue(n *node, dist float64) {
	if _, isInserted := s.insertedInQueue[n.v.ID()]; isInserted {
		return
	}
	s.insertedInQueue[n.v.ID()] = struct{}{}

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
		result[i] = vd.v
	}

	return result
}

func (s *searcher) getResultNodes() []*node {
	return s.resultQueue.getSlice()
}
