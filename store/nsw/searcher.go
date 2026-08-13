package nsw

import (
	"errors"
	"fmt"

	"github.com/FluffyKebab/vektor"
)

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
