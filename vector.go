package vektor

type Vector interface {
	ID() int
	Read() ([]float64, error)
}

type InMemoryVector struct {
	id   int
	data []float64
}

var _ Vector = InMemoryVector{}

func NewInMemoryVector(id int, data []float64) InMemoryVector {
	return InMemoryVector{id, data}
}

func (v InMemoryVector) ID() int {
	return v.id
}

func (v InMemoryVector) Read() ([]float64, error) {
	return v.data, nil
}
