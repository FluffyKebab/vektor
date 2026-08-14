package nsw

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNode(t *testing.T) {
	// Helper to create a clean node with a specific capacity
	createNode := func(capacity int) *node {
		neighbors := make([]nodeDistance, capacity)
		for i := range neighbors {
			neighbors[i].nodeID = emptyNodeID
		}
		return &node{
			nodeStoreID:   1,
			vectorStoreID: 100,
			neighbors:     neighbors,
		}
	}

	t.Run("numNeighbors counts correctly", func(t *testing.T) {
		n := createNode(3)
		assert.Equal(t, 0, n.numNeighbors())

		n.neighbors[0] = nodeDistance{nodeID: 10, dist: 1.0}
		assert.Equal(t, 1, n.numNeighbors())

		n.neighbors[1] = nodeDistance{nodeID: 20, dist: 2.0}
		assert.Equal(t, 2, n.numNeighbors())
	})

	t.Run("addNeighbor returns correct overflow boolean", func(t *testing.T) {
		n := createNode(3)

		// Add 1st (0 indices filled, place+1 = 1 < 3) -> false
		needToRemove := n.addNeighbor(nodeDistance{nodeID: 10, dist: 1.0})
		assert.False(t, needToRemove)
		assert.Equal(t, 1, n.numNeighbors())

		// Add 2nd (1 indices filled, place+1 = 2 < 3) -> false
		needToRemove = n.addNeighbor(nodeDistance{nodeID: 20, dist: 2.0})
		assert.False(t, needToRemove)
		assert.Equal(t, 2, n.numNeighbors())

		// Add 3rd (2 indices filled, place+1 = 3 >= 3) -> true
		needToRemove = n.addNeighbor(nodeDistance{nodeID: 30, dist: 3.0})
		assert.True(t, needToRemove, "Should signal removal needed when reaching capacity")
		assert.Equal(t, 3, n.numNeighbors())

		// Add 4th (3 indices filled, place >= 3) -> true, and ignores insert
		needToRemove = n.addNeighbor(nodeDistance{nodeID: 40, dist: 4.0})
		assert.True(t, needToRemove)
		assert.Equal(t, 3, n.numNeighbors())
		// Ensure 40 wasn't added
		assert.Equal(t, uint32(30), n.neighbors[2].nodeID)
	})

	t.Run("indexOfNeighborWithHighestDistance", func(t *testing.T) {
		n := createNode(5)

		// Empty node
		assert.Equal(t, -1, n.indexOfNeighborWithHighestDistance())

		n.addNeighbor(nodeDistance{nodeID: 10, dist: 2.5})
		n.addNeighbor(nodeDistance{nodeID: 20, dist: 7.1})
		n.addNeighbor(nodeDistance{nodeID: 30, dist: 1.2})
		n.addNeighbor(nodeDistance{nodeID: 40, dist: 7.1}) // Tie for highest

		// Should find index 1 (node 20 with dist 7.1)
		idx := n.indexOfNeighborWithHighestDistance()
		assert.Equal(t, 1, idx)
	})

	t.Run("indexOfNeighbor", func(t *testing.T) {
		n := createNode(3)
		n.addNeighbor(nodeDistance{nodeID: 10, dist: 1.0})
		n.addNeighbor(nodeDistance{nodeID: 20, dist: 2.0})

		assert.Equal(t, 0, n.indexOfNeighbor(10))
		assert.Equal(t, 1, n.indexOfNeighbor(20))
		assert.Equal(t, -1, n.indexOfNeighbor(30), "Should return -1 for non-existent neighbor")
		assert.Equal(t, -1, n.indexOfNeighbor(emptyNodeID), "Should never find emptyNodeID")
	})

	t.Run("removeNeighbor swaps with last and clears", func(t *testing.T) {
		n := createNode(4)
		n.addNeighbor(nodeDistance{nodeID: 10, dist: 1.0}) // idx 0
		n.addNeighbor(nodeDistance{nodeID: 20, dist: 2.0}) // idx 1
		n.addNeighbor(nodeDistance{nodeID: 30, dist: 3.0}) // idx 2

		// Remove middle neighbor (idx 1, node 20)
		n.removeNeighbor(1)

		assert.Equal(t, 2, n.numNeighbors())

		// Node 30 should have swapped into idx 1
		assert.Equal(t, uint32(10), n.neighbors[0].nodeID)
		assert.Equal(t, uint32(30), n.neighbors[1].nodeID)

		// Old last index should be cleared
		assert.Equal(t, emptyNodeID, n.neighbors[2].nodeID)
	})
}

func TestNodeStore(t *testing.T) {
	t.Run("newNode creates correctly initialized nodes", func(t *testing.T) {
		maxN := 3
		store := newNodeStore(maxN)

		assert.Equal(t, 0, store.numNodes())

		// Create first node
		n0 := store.newNode(100) // vectorStoreID = 100
		require.NotNil(t, n0)
		assert.Equal(t, uint32(0), n0.nodeStoreID)
		assert.Equal(t, uint32(100), n0.vectorStoreID)
		assert.Len(t, n0.neighbors, maxN)
		assert.Equal(t, emptyNodeID, n0.neighbors[0].nodeID)
		assert.Equal(t, emptyNodeID, n0.neighbors[maxN-1].nodeID)
		assert.Equal(t, 1, store.numNodes())

		// Create second node
		n1 := store.newNode(200) // vectorStoreID = 200
		require.NotNil(t, n1)
		assert.Equal(t, uint32(1), n1.nodeStoreID)
		assert.Equal(t, uint32(200), n1.vectorStoreID)
		assert.Len(t, n1.neighbors, maxN)
		assert.Equal(t, 2, store.numNodes())

		// Verify underlying flat slices grew correctly
		assert.Len(t, store.neighbors, maxN*2)
		assert.Len(t, store.vectorIDs, 2)
	})

	t.Run("get returns correct node slice views", func(t *testing.T) {
		maxN := 2
		store := newNodeStore(maxN)

		store.newNode(10) // nodeStoreID 0
		store.newNode(20) // nodeStoreID 1

		// Get valid node 0
		n0, ok := store.get(0)
		assert.True(t, ok)
		assert.Equal(t, uint32(10), n0.vectorStoreID)
		assert.Len(t, n0.neighbors, 2)

		// Get valid node 1
		n1, ok := store.get(1)
		assert.True(t, ok)
		assert.Equal(t, uint32(20), n1.vectorStoreID)
		assert.Len(t, n1.neighbors, 2)

		// Modify n0's neighbor view and ensure it affects the flat slice
		n0.neighbors[0] = nodeDistance{nodeID: 99, dist: 5.0}
		assert.Equal(t, uint32(99), store.neighbors[0].nodeID, "Node view should map to backing flat slice")
		assert.Equal(t, emptyNodeID, store.neighbors[2].nodeID, "Node 1 should remain unaffected")

		// Get invalid node out of bounds
		_, ok = store.get(2)
		assert.False(t, ok, "Should return false for out-of-bounds ID")
	})

	t.Run("addNeighbor modifies the underlying flat slice in nodeStore", func(t *testing.T) {
		maxN := 3
		store := newNodeStore(maxN)

		// Create two nodes
		// nodeA will occupy indices 0, 1, 2 in the backing flat slice
		// nodeB will occupy indices 3, 4, 5 in the backing flat slice
		nodeA := store.newNode(100)
		nodeB := store.newNode(200)

		// Add neighbor to nodeA
		nodeA.addNeighbor(nodeDistance{nodeID: nodeB.nodeStoreID, dist: 10.0})

		// 1. Verify nodeA's local slice view is updated
		assert.Equal(t, nodeB.nodeStoreID, nodeA.neighbors[0].nodeID)
		assert.Equal(t, float32(10.0), nodeA.neighbors[0].dist)

		// 2. Verify the underlying store's flat slice is updated at the correct index (0)
		assert.Equal(t, nodeB.nodeStoreID, store.neighbors[0].nodeID)
		assert.Equal(t, float32(10.0), store.neighbors[0].dist)

		// Add neighbor to nodeB to test the offset logic
		nodeB.addNeighbor(nodeDistance{nodeID: nodeA.nodeStoreID, dist: 5.0})

		// 3. Verify the underlying flat slice is updated at the correct offset index (maxN + 0 = 3)
		assert.Equal(t, nodeA.nodeStoreID, store.neighbors[3].nodeID)
		assert.Equal(t, float32(5.0), store.neighbors[3].dist)
	})
}
