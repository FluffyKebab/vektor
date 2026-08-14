package vektor

import "fmt"

type Vector interface {
	ID() uint32
	Read() ([]float64, error)
}

type InMemoryVector struct {
	id   uint32
	data []float64
}

var _ Vector = InMemoryVector{}

func NewInMemoryVector(id uint32, data []float64) InMemoryVector {
	return InMemoryVector{id, data}
}

func (v InMemoryVector) ID() uint32 {
	return v.id
}

func (v InMemoryVector) Read() ([]float64, error) {
	return v.data, nil
}

type VectorStore interface {
	Get(uint32) ([]float64, bool)
	Store([]float64) (uint32, error)
}

type InMemoryVectorStore struct {
	data       []float64
	vectorSize int
	nextID     uint32
}

var _ VectorStore = &InMemoryVectorStore{}

func NewInMemoryVectorStore(vectorSize int) *InMemoryVectorStore {
	return &InMemoryVectorStore{
		make([]float64, 0),
		vectorSize,
		0,
	}
}

func (s *InMemoryVectorStore) Get(id uint32) ([]float64, bool) {
	start := int(id) * s.vectorSize
	end := start + s.vectorSize

	if end > len(s.data) {
		return nil, false
	}

	return s.data[start:end], true
}

func (s *InMemoryVectorStore) Store(v []float64) (uint32, error) {
	if len(v) != s.vectorSize {
		return 0, fmt.Errorf("vector size mismatch: expected %d, got %d", s.vectorSize, len(v))
	}

	id := s.nextID
	s.nextID++

	s.data = append(s.data, v...)
	return id, nil
}
