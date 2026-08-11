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
	maxNeighbors  int
	initNeighbors int
	dist          vektor.Distance
}

var _ vektor.Database = &NSW{}

func (n *NSW) Insert(vec vektor.Vector) error {
	node := newNode(vec, n.maxNeighbors)

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
			removeWorstNeighbor(newNeighbor)
		}
	}

	return nil
}

func removeWorstNeighbor(n *node) {
	if len(n.neighbors) == 0 {
		return
	}

	worstIdx := 0
	maxDist := n.neighbors[0].dist
	for i := 1; i < len(n.neighbors); i++ {
		if n.neighbors[i].dist > maxDist {
			maxDist = n.neighbors[i].dist
			worstIdx = i
		}
	}

	lastIdx := len(n.neighbors) - 1

	n.neighbors[worstIdx] = n.neighbors[lastIdx]

	n.neighbors[lastIdx] = nodeDistance{}
	n.neighbors = n.neighbors[:lastIdx]
}

func (n *NSW) FindClosest(vec vektor.Vector, k int) ([]vektor.Vector, error) {
	s := doSearch(vec, k, n.dist, n.entryNode)
	return s.getResultVectors(), s.getErr()
}

type searcher struct {
	targetVec       vektor.Vector
	candidateQueue  *priorityQueue
	resultQueue     *priorityQueue
	insertedInQueue map[int]struct{}
	dist            vektor.Distance
	distances       map[int]float64
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
		distances:       make(map[int]float64),
		insertedInQueue: make(map[int]struct{}),
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
