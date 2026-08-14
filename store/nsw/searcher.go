package nsw

import (
	"errors"
	"fmt"

	"github.com/FluffyKebab/vektor"
)

type searcher struct {
	targetVec       []float64
	candidateQueue  *candidateQueue
	resultQueue     *priorityQueue
	insertedInQueue map[uint32]struct{}
	distances       map[uint32]float64
	dist            vektor.Distance
	vectors         vektor.VectorStore
	nodes           *nodeStore
	k               int
	ef              int
	err             error
}

func newSearcher(targetVec vektor.Vector, k int, n *NSW) *searcher {
	efSearch := n.efSearch
	if n.efSearch < k {
		efSearch = k
	}

	vecData, err := targetVec.Read()

	s := &searcher{
		targetVec:       vecData,
		dist:            n.dist,
		vectors:         n.vectors,
		nodes:           n.nodes,
		k:               k,
		ef:              efSearch,
		candidateQueue:  n.candidateQueuePool.Get().(*candidateQueue),
		resultQueue:     n.resultQueuePool.Get().(*priorityQueue),
		distances:       n.distancesPool.Get().(map[uint32]float64),
		insertedInQueue: n.insertedPool.Get().(map[uint32]struct{}),
	}

	if err != nil {
		s.err = err
		return s
	}

	entryNode, ok := s.nodes.get(n.entryNodeID)
	if !ok {
		s.err = errors.New("entry node is missing from vector store")
		return s
	}

	entryNodeVec, ok := s.vectors.Get(entryNode.vectorStoreID)
	if !ok {
		s.err = errors.New("entry node is missing from vector store")
		return s
	}

	entryNodeDist, err := s.dist(s.targetVec, entryNodeVec)
	if err != nil {
		s.err = err
		return s
	}
	s.insertedInQueue[n.entryNodeID] = struct{}{}

	s.insertInQueue(entryNode, entryNodeDist)
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

	var curWorstResult *nodeDistance
	if s.resultQueue.len() != 0 {
		worst, err := s.resultQueue.getBottom()
		if err != nil {
			s.err = err
			return false
		}

		curWorstResult = &worst
		if curWorstResult.dist < curBestCandidate.dist {
			return false
		}
	}

	bestCandidateNode, ok := s.nodes.get(curBestCandidate.nodeID)
	if !ok {
		s.err = errors.New("node unexpectedly missing from node store")
		return false
	}

	for _, n := range bestCandidateNode.neighbors {
		if n.nodeID == emptyNodeID {
			break
		}

		s.handleNeighbor(n.nodeID, curWorstResult)
		if s.err != nil {
			return false
		}
	}

	return true
}

func (s *searcher) handleNeighbor(neighborID uint32, curWorstResult *nodeDistance) {
	if _, isInserted := s.insertedInQueue[neighborID]; isInserted {
		return
	}
	s.insertedInQueue[neighborID] = struct{}{}

	nNode, ok := s.nodes.get(neighborID)
	if !ok {
		s.err = errors.New("node in graph is missing from node store")
		return
	}

	nVec, ok := s.vectors.Get(nNode.vectorStoreID)
	if !ok {
		s.err = errors.New("vector in graph is missing from vector store")
		return
	}

	dist, err := s.dist(s.targetVec, nVec)
	if err != nil {
		s.err = fmt.Errorf("calculating distance: %w", err)
		return
	}

	if s.resultQueue.len() < s.ef ||
		(curWorstResult != nil && dist < curWorstResult.dist) {

		s.insertInQueue(nNode, dist)
	}
}

func (s *searcher) insertInQueue(n node, dist float32) {
	elem := nodeDistance{n.nodeStoreID, dist}

	s.resultQueue.insert(elem)
	if s.resultQueue.len() > s.ef {
		s.resultQueue.removeBottom()
	}

	s.candidateQueue.insert(elem)
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
		curNodeID := s.resultQueue.vecs[i].nodeID

		resultNode, ok := s.nodes.get(curNodeID)
		if !ok {
			s.err = errors.New("node in graph is missing from node store")
			return nil
		}

		vec, ok := s.vectors.Get(resultNode.vectorStoreID)
		if !ok {
			s.err = errors.New("vector in graph is missing from vector store")
			return nil
		}

		result = append(result, vektor.NewInMemoryVector(curNodeID, vec))
	}

	return result
}

func (s *searcher) getResultNodeDist() []nodeDistance {
	result := make([]nodeDistance, 0, s.k)
	for i := 0; i < s.k && i < len(s.resultQueue.vecs); i++ {
		result = append(result, s.resultQueue.vecs[i])
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
	n.candidateQueuePool.Put(s.candidateQueue)

	clear(s.resultQueue.vecs)
	s.resultQueue.vecs = s.resultQueue.vecs[:0]
	n.resultQueuePool.Put(s.resultQueue)
}
