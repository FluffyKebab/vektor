package graph

import (
	"errors"
	"fmt"

	"github.com/FluffyKebab/vektor"
)

type searcher struct {
	candidateQueue *HeapQueue
	resultQueue    *SliceQueue
	visited        *VisitedSet
	dist           vektor.Distance
	vectors        vektor.VectorStore
	nodes          NodeStore

	targetVec     []float32
	k             int
	ef            int
	maxIterations int
	err           error
}

func DoGreedySearch(
	targetVec vektor.Vector,
	entryNodeID uint32,
	k,
	ef int,
	maxIterations int,
	candidateQueue *HeapQueue,
	resultQueue *SliceQueue,
	visited *VisitedSet,
	dist vektor.Distance,
	vectors vektor.VectorStore,
	nodes NodeStore,
) ([]NodeDistance, error) {
	if ef < k {
		ef = k
	}

	vecData, err := targetVec.Read()
	if err != nil {
		return nil, err
	}

	s := &searcher{
		targetVec:      vecData,
		dist:           dist,
		vectors:        vectors,
		nodes:          nodes,
		k:              k,
		ef:             ef,
		maxIterations:  maxIterations,
		candidateQueue: candidateQueue,
		resultQueue:    resultQueue,
		visited:        visited,
	}
	s.visited.Update(s.nodes.Len())

	s.doSearch(entryNodeID)
	return s.getResultNodeDist(), s.getErr()
}

func (s *searcher) doSearch(entryNodeID uint32) {
	entryNode, entryVector := s.getNodeAndVector(entryNodeID)
	if s.err != nil {
		return
	}

	entryNodeDist, err := s.dist(s.targetVec, entryVector)
	if err != nil {
		s.err = err
		return
	}

	s.visited.Visit(entryNodeID)
	s.insertInQueue(entryNode, entryNodeDist)

	for i := 0; i < s.maxIterations && s.next(); i++ {
	}
}

func (s *searcher) next() bool {
	if s.candidateQueue.Len() == 0 {
		return false
	}
	curBestCandidate, _ := s.candidateQueue.PopTop()

	curWorstResult := EmptyNodeDistance
	if s.resultQueue.Len() != 0 {
		curWorstResult, _ = s.resultQueue.GetBottom()

		if curWorstResult.Dist < curBestCandidate.Dist {
			return false
		}
	}

	bestCandidateNode, ok := s.nodes.Get(curBestCandidate.NodeID)
	if !ok {
		s.err = errors.New("node unexpectedly missing from node store")
		return false
	}

	for _, n := range bestCandidateNode.Neighbors {
		if n.NodeID == EmptyNodeID {
			break
		}

		s.handleNeighbor(n.NodeID, curWorstResult)
		if s.err != nil {
			return false
		}
	}

	return true
}

func (s *searcher) handleNeighbor(neighborID uint32, curWorstResult NodeDistance) {
	if s.visited.IsVisited(neighborID) {
		return
	}
	s.visited.Visit(neighborID)

	node, vector := s.getNodeAndVector(neighborID)
	if s.err != nil {
		return
	}

	dist, err := s.dist(s.targetVec, vector)
	if err != nil {
		s.err = fmt.Errorf("calculating distance: %w", err)
		return
	}

	if s.resultQueue.Len() < s.ef ||
		(curWorstResult.NodeID != EmptyNodeID && dist < curWorstResult.Dist) {

		s.insertInQueue(node, dist)
	}
}

func (s *searcher) insertInQueue(n Node, dist float32) {
	elem := NodeDistance{NodeID: n.NodeID, Dist: dist}

	s.resultQueue.Insert(elem)
	if s.resultQueue.Len() > s.ef {
		s.resultQueue.PopBottom()
	}

	s.candidateQueue.Insert(elem)
}

func (s *searcher) getNodeAndVector(nodeID uint32) (Node, []float32) {
	node, ok := s.nodes.Get(nodeID)
	if !ok {
		s.err = errors.New("node in graph is missing from node store")
		return Node{}, nil
	}

	vector, ok := s.vectors.Get(node.VectorID)
	if !ok {
		s.err = errors.New("vector in graph is missing from vector store")
		return Node{}, nil
	}

	return node, vector
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

func (s *searcher) getResultNodeDist() []NodeDistance {
	elems := s.resultQueue.Elems()
	result := make([]NodeDistance, 0, s.resultQueue.Len())

	for i := 0; i < s.k && i < len(elems); i++ {
		result = append(result, elems[i])
	}

	return result
}
