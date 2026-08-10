package vektor

type Vector interface {
	ID() int
	Read() ([]float64, error)
}

type InMemoryVector struct {
	id   int
	data []float64
}

func (v InMemoryVector) ID() int {
	return v.id
}

var _ Vector = new(InMemoryVector)

func (v InMemoryVector) Read() ([]float64, error) {
	return v.data, nil
}
