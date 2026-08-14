package nsw

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPriorityQueue_EmptyQueue(t *testing.T) {
	q := newPriorityQueue(50)

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
}

func TestPriorityQueue_InsertAndOrder(t *testing.T) {
	q := newPriorityQueue(50)

	// Insert out of order
	q.insert(nodeDistance{nodeID: 1, dist: 5.0})
	q.insert(nodeDistance{nodeID: 2, dist: 1.0})
	q.insert(nodeDistance{nodeID: 3, dist: 10.0})
	q.insert(nodeDistance{nodeID: 4, dist: 2.5})

	assert.Equal(t, 4, q.len())

	// Validate internal ordering by distance: 1.0, 2.5, 5.0, 10.0
	expectedDistances := []float32{1.0, 2.5, 5.0, 10.0}
	for i, vd := range q.vecs {
		assert.Equal(t, expectedDistances[i], vd.dist)
	}

	// Verify getTop and getBottom return correct extremes without mutating
	top, err := q.getTop()
	assert.NoError(t, err)
	assert.Equal(t, uint32(2), top.nodeID, "getTop should return the node with distance 1.0")

	bottom, err := q.getBottom()
	assert.NoError(t, err)
	assert.Equal(t, uint32(3), bottom.nodeID, "getBottom should return the node with distance 10.0")
}

func TestPriorityQueue_Pops(t *testing.T) {
	q := newPriorityQueue(50)

	q.insert(nodeDistance{nodeID: 1, dist: 2.0})
	q.insert(nodeDistance{nodeID: 2, dist: 1.0})
	q.insert(nodeDistance{nodeID: 3, dist: 3.0})

	// Current order: node 2 (1.0), node 1 (2.0), node 3 (3.0)

	// Test popTop (should be 1.0)
	top, err := q.popTop()
	assert.NoError(t, err)
	assert.Equal(t, uint32(2), top.nodeID)
	assert.Equal(t, 2, q.len(), "Queue length should decrease after popTop")

	// Test popBottom (should be 3.0)
	bottom, err := q.popBottom()
	assert.NoError(t, err)
	assert.Equal(t, uint32(3), bottom.nodeID)
	assert.Equal(t, 1, q.len(), "Queue length should decrease after popBottom")

	// Only node 1 (2.0) should be left
	lastTop, err := q.getTop()
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), lastTop.nodeID)

	lastBottom, err := q.getBottom()
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), lastBottom.nodeID)
}

func TestPriorityQueue_Removes(t *testing.T) {
	q := newPriorityQueue(50)

	q.insert(nodeDistance{nodeID: 1, dist: 1.0})
	q.insert(nodeDistance{nodeID: 2, dist: 2.0})
	q.insert(nodeDistance{nodeID: 3, dist: 3.0})

	// Remove top (1.0)
	q.removeTop()
	assert.Equal(t, 2, q.len())
	top, _ := q.getTop()
	assert.Equal(t, uint32(2), top.nodeID, "Top should now be the 2.0 node")

	// Remove bottom (3.0)
	q.removeBottom()
	assert.Equal(t, 1, q.len())
	bottom, _ := q.getBottom()
	assert.Equal(t, uint32(2), bottom.nodeID, "Bottom should now be the 2.0 node")
}

func TestPriorityQueue_SingleElement(t *testing.T) {
	q := newPriorityQueue(50)

	q.insert(nodeDistance{nodeID: 99, dist: 5.5})
	assert.Equal(t, 1, q.len())

	// Should successfully remove the only element without panicking on slice bounds
	q.removeTop()
	assert.Equal(t, 0, q.len())

	// Insert again and test removeBottom on single element
	q.insert(nodeDistance{nodeID: 99, dist: 5.5})
	q.removeBottom()
	assert.Equal(t, 0, q.len())
}

func TestPriorityQueue_DuplicateDistances(t *testing.T) {
	q := newPriorityQueue(50)

	// Insert items with the exact same distance
	q.insert(nodeDistance{nodeID: 1, dist: 2.0})
	q.insert(nodeDistance{nodeID: 2, dist: 2.0})
	q.insert(nodeDistance{nodeID: 3, dist: 2.0})

	assert.Equal(t, 3, q.len(), "Queue should hold all duplicates")

	// Values should all be 2.0
	top, _ := q.getTop()
	assert.Equal(t, float32(2.0), top.dist)

	bottom, _ := q.getBottom()
	assert.Equal(t, float32(2.0), bottom.dist)

	// Popping should correctly decrement size
	q.popTop()
	assert.Equal(t, 2, q.len())
}

func TestPriorityQueue_CapacityOverflow(t *testing.T) {
	// Initialize with a tiny capacity
	q := newPriorityQueue(2)

	// Insert more elements than the initial capacity
	assert.NotPanics(t, func() {
		q.insert(nodeDistance{nodeID: 1, dist: 10.0})
		q.insert(nodeDistance{nodeID: 2, dist: 5.0})
		q.insert(nodeDistance{nodeID: 3, dist: 20.0})
		q.insert(nodeDistance{nodeID: 4, dist: 1.0})
	}, "Queue should dynamically grow if capacity is exceeded")

	assert.Equal(t, 4, q.len())

	top, _ := q.getTop()
	assert.Equal(t, uint32(4), top.nodeID, "Min element should still be tracked correctly after reallocation")
}
