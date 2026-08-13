package vektor

type Database interface {
	Insert(vec Vector) error
	FindClosest(vec Vector, n int) ([]Vector, error)
}

type GraphObserver interface {
	OnNodeAdded(nodeID uint32)
	OnEdgeAdded(fromID, toID uint32, dist float64)
	OnEdgeRemoved(fromID, toID uint32)
}

type GraphMutater interface {
	AddNode(nodeID uint32) error
	AddEdge(fromID, toID uint32, dist float64) error
	RemoveEdge(fromID, toID uint32) error
}
