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

		assert.Equal(t, 0, store.Len())

		// Create first Node
		n0 := store.newNode(100) // vectorStoreID = 100
		require.NotNil(t, n0)
		assert.Equal(t, uint32(0), n0.NodeID)
		assert.Equal(t, uint32(100), n0.VectorID)
		assert.Len(t, n0.Neighbors, maxN)
		assert.Equal(t, graph.EmptyNodeID, n0.Neighbors[0].NodeID)
		assert.Equal(t, graph.EmptyNodeID, n0.Neighbors[maxN-1].NodeID)
		assert.Equal(t, 1, store.Len())

		// Create second Node
		n1 := store.newNode(200) // vectorStoreID = 100
		require.NotNil(t, n1)
		assert.Equal(t, uint32(1), n1.NodeID)
		assert.Equal(t, uint32(200), n1.VectorID)
		assert.Len(t, n1.Neighbors, maxN)
		assert.Equal(t, 2, store.Len())

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
		n0, err := store.Get(0)
		assert.NoError(t, err)
		assert.Equal(t, uint32(10), n0.VectorID)
		assert.Len(t, n0.Neighbors, 2)

		// Get valid Node 1
		n1, err := store.Get(1)
		assert.NoError(t, err)
		assert.Equal(t, uint32(20), n1.VectorID)
		assert.Len(t, n1.Neighbors, 2)

		// Modify n0's neighbor view and ensure it affects the flat slice
		n0.Neighbors[0] = graph.NodeDistance{NodeID: 99, Dist: 5.0}
		assert.Equal(t, uint32(99), store.neighbors[0].NodeID, "Node view should map to backing flat slice")
		assert.Equal(t, graph.EmptyNodeID, store.neighbors[2].NodeID, "Node 1 should remain unaffected")

		// Get invalid Node out of bounds
		_, err = store.Get(2)
		assert.Error(t, err, "Should get error for out-of-bounds ID")
	})
}
