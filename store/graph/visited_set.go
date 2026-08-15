package graph

type VisitedSet struct {
	data       []uint16
	generation uint16
}

func NewVisitedSet(size int) *VisitedSet {
	return &VisitedSet{
		data:       make([]uint16, size),
		generation: 1,
	}
}

func (s *VisitedSet) Update(numNodes int) {
	s.generation++

	if len(s.data) < numNodes {
		newData := make([]uint16, numNodes)
		copy(newData, s.data)
		s.data = newData
	}

	if s.generation == 0 {
		clear(s.data)
		s.generation = 1
	}
}

func (s *VisitedSet) IsVisited(id uint32) bool {
	if int(id) >= len(s.data) {
		return true
	}

	return s.data[int(id)] == s.generation
}

func (s *VisitedSet) Visit(id uint32) {
	if int(id) >= len(s.data) {
		return
	}

	s.data[int(id)] = s.generation
}
