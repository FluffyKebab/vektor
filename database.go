package vektor

type Database interface {
	Insert(vec Vector) error
	FindClosest(vec Vector, n int) ([]Vector, error)
}
