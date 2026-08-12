package vektor

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
	Get(uint32) (Vector, bool)
	Store(Vector)
}

type InMemoryVectorStore struct {
	data map[uint32]Vector
}

var _ InMemoryVectorStore = InMemoryVectorStore{}

func NewInMemoryVectorStore() *InMemoryVectorStore {
	return &InMemoryVectorStore{
		make(map[uint32]Vector),
	}
}

func (s *InMemoryVectorStore) Get(id uint32) (Vector, bool) {
	vec, found := s.data[id]
	return vec, found
}

func (s *InMemoryVectorStore) Store(v Vector) {
	s.data[v.ID()] = v
}
