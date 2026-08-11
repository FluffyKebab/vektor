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
