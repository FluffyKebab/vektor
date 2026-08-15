package nsw

import (
	"testing"

	"github.com/FluffyKebab/vektor/store/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeStore(t *testing.T) {
	t.Run("newNode creates correctly initialized Nodes", func(t *testing.T) {
		maxN := 3
		store := newNodeStore(maxN)

		assert.Equal(t, 0, store.numNodes())

		// Create first Node
		n0 := store.newNode(100) // vectorStoreID = 100
		require.NotNil(t, n0)
		assert.Equal(t, uint32(0), n0.nodeID)
		assert.Equal(t, uint32(100), n0.vectorID)
		assert.Len(t, n0.neighbors, maxN)
		assert.Equal(t, graph.EmptyNodeID, n0.neighbors[0].NodeID)
		assert.Equal(t, graph.EmptyNodeID, n0.neighbors[maxN-1].NodeID)
		assert.Equal(t, 1, store.numNodes())

		// Create second Node
		n1 := store.newNode(200) // vectorStoreID = 200
		require.NotNil(t, n1)
		assert.Equal(t, uint32(1), n1.nodeID)
		assert.Equal(t, uint32(200), n1.vectorID)
		assert.Len(t, n1.neighbors, maxN)
		assert.Equal(t, 2, store.numNodes())

		// Verify underlying flat slices grew correctly
		assert.Len(t, store.neighbors, maxN*2)
		assert.Len(t, store.vectorIDs, 2)
	})

	t.Run("get returns correct Node slice views", func(t *testing.T) {
		maxN := 2
		store := newNodeStore(maxN)

		store.newNode(10) // NodeStoreID 0
		store.newNode(20) // NodeStoreID 1

		// Get valid Node 0
		n0, ok := store.get(0)
		assert.True(t, ok)
		assert.Equal(t, uint32(10), n0.vectorID)
		assert.Len(t, n0.neighbors, 2)

		// Get valid Node 1
		n1, ok := store.get(1)
		assert.True(t, ok)
		assert.Equal(t, uint32(20), n1.vectorID)
		assert.Len(t, n1.neighbors, 2)

		// Modify n0's neighbor view and ensure it affects the flat slice
		n0.neighbors[0] = graph.NodeDistance{NodeID: 99, Dist: 5.0}
		assert.Equal(t, uint32(99), store.neighbors[0].NodeID, "Node view should map to backing flat slice")
		assert.Equal(t, graph.EmptyNodeID, store.neighbors[2].NodeID, "Node 1 should remain unaffected")

		// Get invalid Node out of bounds
		_, ok = store.get(2)
		assert.False(t, ok, "Should return false for out-of-bounds ID")
	})
}
