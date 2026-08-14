package nsw

import (
	"cmp"
	"errors"
	"slices"
)

// Queue implemented with sorted slice.
type priorityQueue struct {
	vecs []nodeDistance
}

func newPriorityQueue(cap int) *priorityQueue {
	return &priorityQueue{
		vecs: make([]nodeDistance, 0, cap),
	}
}

func (q *priorityQueue) getTop() (nodeDistance, error) {
	if len(q.vecs) == 0 {
		return nodeDistance{}, errors.New("queue is empty")
	}
	return q.vecs[0], nil
}

func (q *priorityQueue) getBottom() (nodeDistance, error) {
	if len(q.vecs) == 0 {
		return nodeDistance{}, errors.New("queue is empty")
	}
	return q.vecs[len(q.vecs)-1], nil
}

func (q *priorityQueue) popTop() (nodeDistance, error) {
	v, err := q.getTop()
	q.removeTop()
	return v, err
}

func (q *priorityQueue) popBottom() (nodeDistance, error) {
	v, err := q.getBottom()
	q.removeBottom()
	return v, err
}

func (q *priorityQueue) removeTop() {
	if len(q.vecs) > 0 {
		copy(q.vecs, q.vecs[1:])
		q.vecs[len(q.vecs)-1] = nodeDistance{}
		q.vecs = q.vecs[:len(q.vecs)-1]
	}
}

func (q *priorityQueue) removeBottom() {
	if len(q.vecs) > 0 {
		q.vecs = q.vecs[:len(q.vecs)-1]
	}
}

func (q *priorityQueue) len() int {
	return len(q.vecs)
}

func (q *priorityQueue) insert(vd nodeDistance) {
	index, _ := slices.BinarySearchFunc(q.vecs, vd, func(a, b nodeDistance) int {
		return cmp.Compare(a.dist, b.dist)
	})

	q.vecs = slices.Insert(q.vecs, index, vd)
}

// Queue implemented with heap.
type candidateQueue struct {
	vecs []nodeDistance
}

func newCandidateQueue(capacity int) *candidateQueue {
	return &candidateQueue{
		vecs: make([]nodeDistance, 0, capacity),
	}
}

func (q *candidateQueue) insert(nd nodeDistance) {
	q.vecs = append(q.vecs, nd)
	q.up(len(q.vecs) - 1)
}

func (q *candidateQueue) popTop() (nodeDistance, error) {
	n := len(q.vecs) - 1
	if n < 0 {
		return nodeDistance{}, errors.New("queue is empty")
	}

	// Swap the top with the bottom
	q.vecs[0], q.vecs[n] = q.vecs[n], q.vecs[0]

	// Shift the new top down to its correct position
	q.down(0, n)

	// Extract the old top (now at the bottom)
	top := q.vecs[n]
	q.vecs = q.vecs[:n]

	return top, nil
}

func (q *candidateQueue) len() int {
	return len(q.vecs)
}

// up shifts an element up the tree until it is smaller than its parent
func (q *candidateQueue) up(j int) {
	for {
		i := (j - 1) / 2 // parent index
		if i == j || q.vecs[j].dist >= q.vecs[i].dist {
			break
		}
		q.vecs[i], q.vecs[j] = q.vecs[j], q.vecs[i]
		j = i
	}
}

// down shifts an element down the tree until it is smaller than its children
func (q *candidateQueue) down(i0, n int) {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child

		// If right child exists and is smaller than left child, use right child
		if j2 := j1 + 1; j2 < n && q.vecs[j2].dist < q.vecs[j1].dist {
			j = j2
		}

		if q.vecs[j].dist >= q.vecs[i].dist {
			break
		}

		q.vecs[i], q.vecs[j] = q.vecs[j], q.vecs[i]
		i = j
	}
}
