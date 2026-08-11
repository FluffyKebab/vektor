package vektor

type Database interface {
	Insert(vec Vector) error
	FindClosest(vec Vector, n int) ([]Vector, error)
}

type GraphObserver interface {
	OnNodeAdded(nodeID uint32)
	OnEdgeAdded(fromID, toID uint32)
	OnEdgeRemoved(fromID, toID uint32)
}
