package graph

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNode(t *testing.T) {
	// Helper to create a clean Node with a specific capacity
	createNode := func(capacity int) Neighbors {
		neighbors := make([]NodeDistance, capacity)
		for i := range neighbors {
			neighbors[i].NodeID = EmptyNodeID
		}
		return neighbors
	}

	t.Run("numNeighbors counts correctly", func(t *testing.T) {
		n := createNode(3)
		require.Equal(t, 0, n.Len())

		n[0] = NodeDistance{NodeID: 10, Dist: 1.0}
		require.Equal(t, 1, n.Len())

		n[1] = NodeDistance{NodeID: 20, Dist: 2.0}
		require.Equal(t, 2, n.Len())
	})

	t.Run("addNeighbor returns correct overflow boolean", func(t *testing.T) {
		n := createNode(3)

		// Add 1st (0 indices filled, place+1 = 1 < 3) -> false
		needToRemove := n.AddNeighbor(NodeDistance{NodeID: 10, Dist: 1.0})
		require.False(t, needToRemove)
		require.Equal(t, 1, n.Len())

		// Add 2nd (1 indices filled, place+1 = 2 < 3) -> false
		needToRemove = n.AddNeighbor(NodeDistance{NodeID: 20, Dist: 2.0})
		require.False(t, needToRemove)
		require.Equal(t, 2, n.Len())

		// Add 3rd (2 indices filled, place+1 = 3 >= 3) -> true
		needToRemove = n.AddNeighbor(NodeDistance{NodeID: 30, Dist: 3.0})
		require.True(t, needToRemove, "Should signal removal needed when reaching capacity")
		require.Equal(t, 3, n.Len())

		// Add 4th (3 indices filled, place >= 3) -> true, and ignores insert
		needToRemove = n.AddNeighbor(NodeDistance{NodeID: 40, Dist: 4.0})
		require.True(t, needToRemove)
		require.Equal(t, 3, n.Len())
		// Ensure 40 wasn't added
		require.Equal(t, uint32(30), n[2].NodeID)
	})

	t.Run("indexOfNeighborWithHighestDistance", func(t *testing.T) {
		n := createNode(5)

		// Empty Node
		require.Equal(t, -1, n.HighestDistanceIndex())

		n.AddNeighbor(NodeDistance{NodeID: 10, Dist: 2.5})
		n.AddNeighbor(NodeDistance{NodeID: 20, Dist: 7.1})
		n.AddNeighbor(NodeDistance{NodeID: 30, Dist: 1.2})
		n.AddNeighbor(NodeDistance{NodeID: 40, Dist: 7.1}) // Tie for highest

		// Should find index 1 (Node 20 with Dist 7.1)
		idx := n.HighestDistanceIndex()
		require.Equal(t, 1, idx)
	})

	t.Run("indexOfNeighbor", func(t *testing.T) {
		n := createNode(3)
		n.AddNeighbor(NodeDistance{NodeID: 10, Dist: 1.0})
		n.AddNeighbor(NodeDistance{NodeID: 20, Dist: 2.0})

		require.Equal(t, 0, n.IndexOf(10))
		require.Equal(t, 1, n.IndexOf(20))
		require.Equal(t, -1, n.IndexOf(30), "Should return -1 for non-existent neighbor")
		require.Equal(t, -1, n.IndexOf(EmptyNodeID), "Should never find emptyNodeID")
	})

	t.Run("removeNeighbor swaps with last and clears", func(t *testing.T) {
		n := createNode(4)
		n.AddNeighbor(NodeDistance{NodeID: 10, Dist: 1.0}) // idx 0
		n.AddNeighbor(NodeDistance{NodeID: 20, Dist: 2.0}) // idx 1
		n.AddNeighbor(NodeDistance{NodeID: 30, Dist: 3.0}) // idx 2

		// Remove middle neighbor (idx 1, Node 20)
		n.Remove(1)

		require.Equal(t, 2, n.Len())

		// Node 30 should have swapped into idx 1
		require.Equal(t, uint32(10), n[0].NodeID)
		require.Equal(t, uint32(30), n[1].NodeID)

		// Old last index should be cleared
		require.Equal(t, EmptyNodeID, n[2].NodeID)
	})
}
