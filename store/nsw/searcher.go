package nsw

import (
	"errors"
	"fmt"

	"github.com/FluffyKebab/vektor"
	"github.com/FluffyKebab/vektor/store/graph"
)

type searcher struct {
	targetVec      []float32
	candidateQueue *graph.HeapQueue
	resultQueue    *graph.SliceQueue
	visited        *graph.VisitedSet
	dist           vektor.Distance
	vectors        vektor.VectorStore
	nodes          *nodeStore
	k              int
	ef             int
	err            error
}

func newSearcher(targetVec vektor.Vector, k, ef int, n *NSW) *searcher {
	if ef < k {
		ef = k
	}

	vecData, err := targetVec.Read()

	s := &searcher{
		targetVec:      vecData,
		dist:           n.dist,
		vectors:        n.vectors,
		nodes:          n.nodes,
		k:              k,
		ef:             ef,
		candidateQueue: n.candidateQueuePool.Get().(*graph.HeapQueue),
		resultQueue:    n.resultQueuePool.Get().(*graph.SliceQueue),
		visited:        n.visited.Get().(*graph.VisitedSet),
	}
	s.visited.Update(s.nodes.numNodes())

	if err != nil {
		s.err = err
		return s
	}

	entryNode, ok := s.nodes.get(n.entryNodeID)
	if !ok {
		s.err = errors.New("entry node is missing from vector store")
		return s
	}

	entryNodeVec, ok := s.vectors.Get(entryNode.vectorID)
	if !ok {
		s.err = errors.New("entry node is missing from vector store")
		return s
	}

	entryNodeDist, err := s.dist(s.targetVec, entryNodeVec)
	if err != nil {
		s.err = err
		return s
	}
	s.visited.Visit(n.entryNodeID)

	s.insertInQueue(entryNode, entryNodeDist)
	return s
}

func doSearch(vec vektor.Vector, k, ef int, n *NSW) *searcher {
	s := newSearcher(vec, k, ef, n)
	for s.next() {
	}

	return s
}

func (s *searcher) next() bool {
	if s.err != nil {
		return false
	}

	if s.candidateQueue.Len() == 0 {
		return false
	}

	curBestCandidate, err := s.candidateQueue.PopTop()
	if err != nil {
		s.err = err
		return false
	}

	curWorstResult := graph.EmptyNodeDistance
	if s.resultQueue.Len() != 0 {
		curWorstResult, _ = s.resultQueue.GetBottom()
		if curWorstResult.Dist < curBestCandidate.Dist {
			return false
		}
	}

	bestCandidateNode, ok := s.nodes.get(curBestCandidate.NodeID)
	if !ok {
		s.err = errors.New("node unexpectedly missing from node store")
		return false
	}

	for _, n := range bestCandidateNode.neighbors {
		if n.NodeID == graph.EmptyNodeID {
			break
		}

		s.handleNeighbor(n.NodeID, curWorstResult)
		if s.err != nil {
			return false
		}
	}

	return true
}

func (s *searcher) handleNeighbor(neighborID uint32, curWorstResult graph.NodeDistance) {
	if s.visited.IsVisited(neighborID) {
		return
	}
	s.visited.Visit(neighborID)

	nNode, ok := s.nodes.get(neighborID)
	if !ok {
		s.err = errors.New("node in graph is missing from node store")
		return
	}

	nVec, ok := s.vectors.Get(nNode.vectorID)
	if !ok {
		s.err = errors.New("vector in graph is missing from vector store")
		return
	}

	dist, err := s.dist(s.targetVec, nVec)
	if err != nil {
		s.err = fmt.Errorf("calculating distance: %w", err)
		return
	}

	if s.resultQueue.Len() < s.ef ||
		(curWorstResult.NodeID != graph.EmptyNodeID && dist < curWorstResult.Dist) {

		s.insertInQueue(nNode, dist)
	}
}

func (s *searcher) insertInQueue(n node, dist float32) {
	elem := graph.NodeDistance{NodeID: n.nodeID, Dist: dist}

	s.resultQueue.Insert(elem)
	if s.resultQueue.Len() > s.ef {
		s.resultQueue.PopBottom()
	}

	s.candidateQueue.Insert(elem)
}

func (s *searcher) getErr() error {
	if s.resultQueue.Len() == 0 {
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
	elems := s.resultQueue.Elems()

	for i := 0; i < s.k && i < len(elems); i++ {
		curNodeID := elems[i].NodeID

		resultNode, ok := s.nodes.get(curNodeID)
		if !ok {
			s.err = errors.New("node in graph is missing from node store")
			return nil
		}

		vec, ok := s.vectors.Get(resultNode.vectorID)
		if !ok {
			s.err = errors.New("vector in graph is missing from vector store")
			return nil
		}

		result = append(result, vektor.NewInMemoryVector(curNodeID, vec))
	}

	return result
}

func (s *searcher) getResultNodeDist() []graph.NodeDistance {
	return s.resultQueue.Elems()
}

func (s *searcher) cleanup(n *NSW) {
	n.visited.Put(s.visited)

	s.candidateQueue.Clear()
	n.candidateQueuePool.Put(s.candidateQueue)

	s.resultQueue.Clear()
	n.resultQueuePool.Put(s.resultQueue)
}
