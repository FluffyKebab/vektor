package vektor

import (
	"cmp"
	"errors"
	"slices"
)

// priorityQueue stores pairs of nodes and distances from lowest to highest distance.
type priorityQueue struct {
	vecs []nodeDistance
}

func newPriorityQueue() *priorityQueue {
	return &priorityQueue{
		vecs: make([]nodeDistance, 0),
	}
}

// getTop returns the Vector with the lowest distance.
func (q *priorityQueue) getTop() (nodeDistance, error) {
	if len(q.vecs) == 0 {
		return nodeDistance{}, errors.New("queue is empty")
	}
	return q.vecs[0], nil
}

// getBottom returns the Vector with the highest distance.
func (q *priorityQueue) getBottom() (nodeDistance, error) {
	if len(q.vecs) == 0 {
		return nodeDistance{}, errors.New("queue is empty")
	}
	return q.vecs[len(q.vecs)-1], nil
}

// popTop removes and returns the node with the lowest distance.
func (q *priorityQueue) popTop() (nodeDistance, error) {
	v, err := q.getTop()
	q.removeTop()
	return v, err
}

// popBottom removes and returns the node with the highest distance.
func (q *priorityQueue) popBottom() (nodeDistance, error) {
	v, err := q.getBottom()
	q.removeBottom()
	return v, err
}

// removeTop drops the lowest distance element by slicing forward.
func (q *priorityQueue) removeTop() {
	if len(q.vecs) > 0 {
		q.vecs[0] = nodeDistance{}
		q.vecs = q.vecs[1:]
	}
}

// removeBottom drops the highest distance element by slicing off the end.
func (q *priorityQueue) removeBottom() {
	if len(q.vecs) > 0 {
		q.vecs = q.vecs[:len(q.vecs)-1]
	}
}

// len returns the number of elements currently in the queue.
func (q *priorityQueue) len() int {
	return len(q.vecs)
}

// insert places a new vectorDistance in the correct sorted position.
func (q *priorityQueue) insert(vd nodeDistance) {
	index, _ := slices.BinarySearchFunc(q.vecs, vd, func(a, b nodeDistance) int {
		return cmp.Compare(a.dist, b.dist)
	})

	q.vecs = slices.Insert(q.vecs, index, vd)
}

// getSlice returns a new slice containing just the Vectors, sorted by distance.
func (q *priorityQueue) getSlice() []*node {
	result := make([]*node, len(q.vecs))
	for i, vd := range q.vecs {
		result[i] = vd.node
	}
	return result
}

func (q *priorityQueue) getElements() []nodeDistance {
	return q.vecs
}
