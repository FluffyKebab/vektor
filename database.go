package vektor

type Database interface {
	Insert(vec Vector) error
	Delete(vec Vector) error
	FindClosest(vec Vector, n int) ([]Vector, error)
}
