package nsw

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPriorityQueue_EmptyQueue(t *testing.T) {
	q := newPriorityQueue()

	assert.Equal(t, 0, q.len(), "New queue should have length 0")

	// Test getters on empty queue
	_, err := q.getTop()
	assert.Error(t, err, "getTop on empty queue should return an error")
	assert.EqualError(t, err, "queue is empty")

	_, err = q.getBottom()
	assert.Error(t, err, "getBottom on empty queue should return an error")
	assert.EqualError(t, err, "queue is empty")

	// Test pops on empty queue
	_, err = q.popTop()
	assert.Error(t, err, "popTop on empty queue should return an error")

	_, err = q.popBottom()
	assert.Error(t, err, "popBottom on empty queue should return an error")

	// These should silently do nothing without panicking
	assert.NotPanics(t, func() {
		q.removeTop()
		q.removeBottom()
	}, "Removing from empty queue should not panic")

	// Test slice conversion on empty queue
	slice := q.getSlice()
	assert.Empty(t, slice, "getSlice on empty queue should return empty slice")
}

func TestPriorityQueue_InsertAndOrder(t *testing.T) {
	q := newPriorityQueue()

	n1 := &node{}
	n2 := &node{}
	n3 := &node{}
	n4 := &node{}
	n5 := &node{} // For testing duplicate distances

	// Insert out of order
	q.insert(nodeDistance{node: n1, dist: 5.0})
	q.insert(nodeDistance{node: n2, dist: 1.0})
	q.insert(nodeDistance{node: n3, dist: 10.0})
	q.insert(nodeDistance{node: n4, dist: 2.5})
	q.insert(nodeDistance{node: n5, dist: 5.0}) // Duplicate

	assert.Equal(t, 5, q.len())

	// Validate internal ordering by distance: 1.0, 2.5, 5.0, 5.0, 10.0
	expectedDistances := []float64{1.0, 2.5, 5.0, 5.0, 10.0}
	for i, vd := range q.vecs {
		assert.Equal(t, expectedDistances[i], vd.dist)
	}

	// Verify getTop and getBottom return correct extremes without mutating
	top, err := q.getTop()
	assert.NoError(t, err)
	assert.Same(t, n2, top.node, "getTop should return the node with distance 1.0")

	bottom, err := q.getBottom()
	assert.NoError(t, err)
	assert.Same(t, n3, bottom.node, "getBottom should return the node with distance 10.0")
}

func TestPriorityQueue_Pops(t *testing.T) {
	q := newPriorityQueue()

	n1 := &node{}
	n2 := &node{}
	n3 := &node{}

	q.insert(nodeDistance{node: n1, dist: 2.0})
	q.insert(nodeDistance{node: n2, dist: 1.0})
	q.insert(nodeDistance{node: n3, dist: 3.0})

	// Current order: n2 (1.0), n1 (2.0), n3 (3.0)

	// Test popTop (should be 1.0)
	top, err := q.popTop()
	assert.NoError(t, err)
	assert.Same(t, n2, top.node)
	assert.Equal(t, 2, q.len(), "Queue length should decrease after popTop")

	// Test popBottom (should be 3.0)
	bottom, err := q.popBottom()
	assert.NoError(t, err)
	assert.Same(t, n3, bottom.node)
	assert.Equal(t, 1, q.len(), "Queue length should decrease after popBottom")

	// Only n1 (2.0) should be left
	lastTop, err := q.getTop()
	assert.NoError(t, err)
	assert.Same(t, n1, lastTop.node)

	lastBottom, err := q.getBottom()
	assert.NoError(t, err)
	assert.Same(t, n1, lastBottom.node)
}

func TestPriorityQueue_Removes(t *testing.T) {
	q := newPriorityQueue()

	n1 := &node{}
	n2 := &node{}
	n3 := &node{}

	q.insert(nodeDistance{node: n1, dist: 1.0})
	q.insert(nodeDistance{node: n2, dist: 2.0})
	q.insert(nodeDistance{node: n3, dist: 3.0})

	// Remove top (1.0)
	q.removeTop()
	assert.Equal(t, 2, q.len())
	top, _ := q.getTop()
	assert.Same(t, n2, top.node, "Top should now be the 2.0 node")

	// Remove bottom (3.0)
	q.removeBottom()
	assert.Equal(t, 1, q.len())
	bottom, _ := q.getBottom()
	assert.Same(t, n2, bottom.node, "Bottom should now be the 2.0 node")
}

func TestPriorityQueue_GetSlice(t *testing.T) {
	q := newPriorityQueue()

	n1 := &node{}
	n2 := &node{}
	n3 := &node{}

	// Insert randomly
	q.insert(nodeDistance{node: n3, dist: 3.0})
	q.insert(nodeDistance{node: n1, dist: 1.0})
	q.insert(nodeDistance{node: n2, dist: 2.0})

	slice := q.getSlice()

	assert.Len(t, slice, 3)

	// Assert order is correctly maintained in the resulting slice of node pointers
	expectedSlice := []*node{n1, n2, n3}
	assert.Equal(t, expectedSlice, slice)
}
