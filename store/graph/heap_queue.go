package graph

// Queue implemented with heap.
type HeapQueue struct {
	elems []NodeDistance
}

func NewHeapQueue(capacity int) *HeapQueue {
	return &HeapQueue{
		elems: make([]NodeDistance, 0, capacity),
	}
}

// Inserts an element into the queue.
func (q *HeapQueue) Insert(nd NodeDistance) {
	q.elems = append(q.elems, nd)
	q.up(len(q.elems) - 1)
}

// PopTop pops the element with the lowest distance. Errors if length of queue is zero.
func (q *HeapQueue) PopTop() (NodeDistance, error) {
	n := len(q.elems) - 1
	if n < 0 {
		return NodeDistance{}, ErrEmptyQueue
	}

	q.elems[0], q.elems[n] = q.elems[n], q.elems[0]
	q.down(0, n)
	top := q.elems[n]
	q.elems = q.elems[:n]

	return top, nil
}

func (q *HeapQueue) Len() int {
	return len(q.elems)
}

func (q *HeapQueue) Clear() {
	clear(q.elems)
	q.elems = q.elems[:0]
}

// up shifts an element up the tree until it is smaller than its parent
func (q *HeapQueue) up(j int) {
	for {
		i := (j - 1) / 2
		if i == j || q.elems[j].Dist >= q.elems[i].Dist {
			break
		}
		q.elems[i], q.elems[j] = q.elems[j], q.elems[i]
		j = i
	}
}

// down shifts an element down the tree until it is smaller than its children
func (q *HeapQueue) down(i0, n int) {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child

		// If right child exists and is smaller than left child, use right child
		if j2 := j1 + 1; j2 < n && q.elems[j2].Dist < q.elems[j1].Dist {
			j = j2
		}

		if q.elems[j].Dist >= q.elems[i].Dist {
			break
		}

		q.elems[i], q.elems[j] = q.elems[j], q.elems[i]
		i = j
	}
}
