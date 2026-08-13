package nsw

type node struct {
	id        uint32
	neighbors []nodeDistance
}

type nodeDistance struct {
	node *node
	dist float64
}

func newNode(id uint32, maxN int) *node {
	return &node{id: id, neighbors: make([]nodeDistance, 0, maxN+1)}
}

// neighborWithHighestDistance assumes that neighbors is not empty.
func (n *node) neighborWithHighestDistance() int {
	worstIdx := 1
	maxDist := n.neighbors[0].dist
	for i := 1; i < len(n.neighbors); i++ {
		if n.neighbors[i].dist > maxDist {
			maxDist = n.neighbors[i].dist
			worstIdx = i
		}
	}

	return worstIdx
}

func (n *node) findNeighborIndex(neighborID uint32) int {
	for i := 0; i < len(n.neighbors); i++ {
		if n.neighbors[i].node.id == neighborID {
			return i
		}
	}

	return -1
}

func (n *node) removeNeighbor(i int) {
	lastIdx := len(n.neighbors) - 1
	n.neighbors[i] = n.neighbors[lastIdx]
	n.neighbors[lastIdx] = nodeDistance{}
	n.neighbors = n.neighbors[:lastIdx]
}
