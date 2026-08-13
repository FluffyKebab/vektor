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
	distances       map[uint32]float64
	dist            vektor.Distance
	store           vektor.VectorStore
	k               int
	ef              int
	err             error
}

func newSearcher(targetVec vektor.Vector, k int, n *NSW) *searcher {
	efSearch := n.efSearch
	if n.efSearch < k {
		efSearch = k
	}

	s := &searcher{
		targetVec:       targetVec,
		dist:            n.dist,
		store:           n.store,
		k:               k,
		ef:              efSearch,
		candidateQueue:  n.queuePool.Get().(*priorityQueue),
		resultQueue:     n.queuePool.Get().(*priorityQueue),
		distances:       n.distancesPool.Get().(map[uint32]float64),
		insertedInQueue: n.insertedPool.Get().(map[uint32]struct{}),
	}

	entryVec, ok := s.store.Get(n.entryNode.id)
	if !ok {
		s.err = errors.New("entry node vector is missing from vectore store")
		return s
	}

	entryNodeDist, err := s.getDist(entryVec, targetVec)
	if err != nil {
		s.err = err
		return s
	}

	s.insertInQueue(n.entryNode, entryNodeDist)
	return s
}

func doSearch(vec vektor.Vector, k int, n *NSW) *searcher {
	s := newSearcher(vec, k, n)
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

	if s.resultQueue.len() <= s.ef ||
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
	if s.resultQueue.len() > s.ef {
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
	result := make([]vektor.Vector, 0, s.k)

	for i := 0; i < s.k && i < len(s.resultQueue.vecs); i++ {
		vec, ok := s.store.Get(s.resultQueue.vecs[i].node.id) // PANIC
		if !ok {
			s.err = errors.New("vector in graph is missing from vector store")
			return nil
		}

		result = append(result, vec)
	}

	return result
}

func (s *searcher) getResultNodes() []*node {
	result := make([]*node, 0, s.k)
	for i := 0; i < s.k && i < len(s.resultQueue.vecs); i++ {
		result = append(result, s.resultQueue.vecs[i].node)
	}

	return result
}

func (s *searcher) cleanup(n *NSW) {
	clear(s.distances)
	n.distancesPool.Put(s.distances)

	clear(s.insertedInQueue)
	n.insertedPool.Put(s.insertedInQueue)

	clear(s.candidateQueue.vecs)
	s.candidateQueue.vecs = s.candidateQueue.vecs[:0]
	n.queuePool.Put(s.candidateQueue)

	clear(s.resultQueue.vecs)
	s.resultQueue.vecs = s.resultQueue.vecs[:0]
	n.queuePool.Put(s.resultQueue)
}
