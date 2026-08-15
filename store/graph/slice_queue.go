package graph

import (
	"cmp"
	"errors"
	"slices"
)

var ErrEmptyQueue = errors.New("queue is empty")

// Queue implemented with an underling sorted slice.
type SliceQueue struct {
	elems []NodeDistance
}

func NewSliceQueue(cap int) *SliceQueue {
	return &SliceQueue{
		elems: make([]NodeDistance, 0, cap),
	}
}

// GetTop returns the element with the lowest distance.
func (q *SliceQueue) GetTop() (NodeDistance, error) {
	if len(q.elems) == 0 {
		return NodeDistance{}, ErrEmptyQueue
	}
	return q.elems[0], nil
}

// GetBottom returns the element with the highest distance. Returns an error if the queue is empty.
func (q *SliceQueue) GetBottom() (NodeDistance, error) {
	if len(q.elems) == 0 {
		return NodeDistance{}, ErrEmptyQueue
	}
	return q.elems[len(q.elems)-1], nil
}

// PopTop pops the element with the lowest distance.
func (q *SliceQueue) PopTop() (NodeDistance, error) {
	v, err := q.GetTop()
	q.removeTop()
	return v, err
}

// PopBottom pops the element with the highest distance.
func (q *SliceQueue) PopBottom() (NodeDistance, error) {
	v, err := q.GetBottom()
	q.removeBottom()
	return v, err
}

func (q *SliceQueue) Elems() []NodeDistance {
	return q.elems
}

func (q *SliceQueue) Clear() {
	clear(q.elems)
	q.elems = q.elems[:0]
}

func (q *SliceQueue) removeTop() {
	if len(q.elems) > 0 {
		copy(q.elems, q.elems[1:])
		q.elems[len(q.elems)-1] = NodeDistance{}
		q.elems = q.elems[:len(q.elems)-1]
	}
}

func (q *SliceQueue) removeBottom() {
	if len(q.elems) > 0 {
		q.elems = q.elems[:len(q.elems)-1]
	}
}

func (q *SliceQueue) Len() int {
	return len(q.elems)
}

func (q *SliceQueue) Insert(vd NodeDistance) {
	index, _ := slices.BinarySearchFunc(q.elems, vd, func(a, b NodeDistance) int {
		return cmp.Compare(a.Dist, b.Dist)
	})
	q.elems = slices.Insert(q.elems, index, vd)
}
