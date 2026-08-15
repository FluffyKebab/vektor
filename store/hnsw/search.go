package hnsw

import (
	"github.com/FluffyKebab/vektor"
)

type searcher struct {
	nodes   *nodeStore
	vectors vektor.VectorStore
	err     error
	target  []float32
	k       int
	ef      int
}

func (s *searcher) next() bool {
	if s.err != nil {
		return false
	}

	/* if s.candidateQueue.len() == 0 {
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
	} */

	return true
}
