package vektor

type node struct {
	v            Vector
	neighbors    []*node
	maxNeighbors int
}

type NSW struct {
	entryNode *node
}

var _ Database = NSW{}

func (n NSW) Insert(vec Vector) error {
	panic("unimplemented")
}

func (n NSW) Delete(vec Vector) error {
	panic("unimplemented")
}

func (n NSW) FindClosest(vec Vector, k int, dist distance) ([]Vector, error) {
	s := newSearcher(vec, k, dist, n.entryNode)
	for s.next() {
	}

	return s.getResult(), s.getErr()
}

type searcher struct {
	targetVec       Vector
	candidateQueue  *priorityQueue
	resultQueue     *priorityQueue
	insertedInQueue map[int]struct{}
	dist            distance
	distances       map[int]float64
	k               int
	err             error
}

func newSearcher(targetVec Vector, k int, dist distance, entryNode *node) *searcher {
	s := &searcher{
		targetVec:       targetVec,
		candidateQueue:  newPriorityQueue(),
		resultQueue:     newPriorityQueue(),
		dist:            dist,
		distances:       make(map[int]float64),
		insertedInQueue: make(map[int]struct{}),
		k:               k,
	}

	entryNodeDist, err := s.getDist(targetVec, entryNode.v)
	if err != nil {
		s.err = err
	}

	s.candidateQueue.insert(nodeDistance{entryNode, entryNodeDist})
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
		s.handleNeighbor(n, curWorstResult)
	}

	return true
}

func (s *searcher) handleNeighbor(neighbor *node, curWorstResult nodeDistance) {
	dist, err := s.getDist(neighbor.v, s.targetVec)
	if err != nil {
		s.err = err
		return
	}

	if s.resultQueue.len() < s.k {
		s.insertInQue(neighbor, dist)
	}

	if curWorstResult.node != nil && dist < curWorstResult.dist {
		s.insertInQue(neighbor, dist)
	}
}

func (s *searcher) insertInQue(n *node, dist float64) {
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

func (n *searcher) getDist(curVec, targetVec Vector) (float64, error) {
	if d, found := n.distances[curVec.ID()]; found {
		return d, nil
	}

	d, err := n.dist(curVec, targetVec)
	if err != nil {
		return 0, err
	}

	n.distances[curVec.ID()] = d
	return d, nil
}

func (s *searcher) getErr() error {
	return s.err
}

func (s *searcher) getResult() []Vector {
	result := make([]Vector, s.resultQueue.len())
	for i, vd := range s.resultQueue.getSlice() {
		result[i] = vd.v
	}

	return result
}
