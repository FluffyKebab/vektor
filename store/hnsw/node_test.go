package hnsw

import (
	"testing"

	"github.com/FluffyKebab/vektor/store/graph"
	"github.com/stretchr/testify/require"
)

func TestNodeStore(t *testing.T) {
	// Helper to create a standard test store: 4 layers, M0=32, M=16, cap=10
	createStore := func() *nodeStore {
		return newNodeStore(4, 32, 16, 10)
	}

	t.Run("Initialization", func(t *testing.T) {
		s := createStore()
		require.Equal(t, 0, s.len())
		require.Equal(t, 4, s.layers)
		require.Equal(t, 32, s.maxN0)
		require.Equal(t, 16, s.maxN1)
		require.Len(t, s.layerLengths, 4)
		require.Empty(t, s.bottomLayerNeighbors)
		require.Empty(t, s.upperLayerNeighbors)
	})

	t.Run("Add node at Layer 0 only", func(t *testing.T) {
		s := createStore()

		nodeID, err := s.add(100, 0)
		require.NoError(t, err)

		require.Equal(t, uint32(0), nodeID)
		require.Equal(t, 1, s.len())

		// Verify Layer 0 allocation (M0 = 32)
		require.Len(t, s.bottomLayerNeighbors, 32)
		require.Equal(t, graph.EmptyNodeID, s.bottomLayerNeighbors[0].NodeID)
		require.Equal(t, graph.EmptyNodeID, s.bottomLayerNeighbors[31].NodeID)

		// Verify Upper Layers were NOT allocated
		require.Len(t, s.upperLayerNeighbors, 0)
		require.Equal(t, uint32(0), s.offsets[0])

		// Verify layer lengths tracker
		require.Equal(t, []int{1, 0, 0, 0}, s.layerLengths)
	})

	t.Run("Add multi-layer node and verify dense/sparse offsets", func(t *testing.T) {
		s := createStore()

		// Node 0: Exists up to Layer 2
		id0, err := s.add(100, 2)
		require.NoError(t, err)
		require.Equal(t, uint32(0), id0)

		// Node 1: Exists up to Layer 1
		id1, err := s.add(200, 1)
		require.NoError(t, err)
		require.Equal(t, uint32(1), id1)

		// Verify allocations
		// Bottom layer: 2 nodes * 32 = 64 slots
		require.Len(t, s.bottomLayerNeighbors, 64)

		// Upper layer: Node 0 has 2 layers (2*16=32), Node 1 has 1 layer (1*16=16) -> Total 48 slots
		require.Len(t, s.upperLayerNeighbors, 48)

		// Verify Offsets for the jagged array
		require.Equal(t, uint32(0), s.offsets[0])  // Node 0 upper layers start at index 0
		require.Equal(t, uint32(32), s.offsets[1]) // Node 1 upper layers start at index 32 (after Node 0's two layers)

		// Verify layer lengths tracker
		// Layer 0: Nodes 0, 1
		// Layer 1: Nodes 0, 1
		// Layer 2: Node 0
		// Layer 3: None
		require.Equal(t, []int{2, 2, 1, 0}, s.layerLengths)
	})

	t.Run("Add node fails for invalid layer", func(t *testing.T) {
		s := createStore()

		// Try to add at layer 4 (Store only has layers 0, 1, 2, 3)
		_, err := s.add(100, 4)
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid layer")
	})

	t.Run("Get node returns correct slice views that mutate backing array", func(t *testing.T) {
		s := createStore()
		s.add(100, 2) // Node 0
		s.add(200, 1) // Node 1

		// Get Node 0 at Layer 0 (Base Layer)
		n0L0, err := s.get(0, 0)
		require.NoError(t, err)
		require.Len(t, n0L0.neighbors, 32)

		// Mutate the view and verify it hits the flat array
		n0L0.neighbors[0] = graph.NodeDistance{NodeID: 999, Dist: 5.5}
		require.Equal(t, uint32(999), s.bottomLayerNeighbors[0].NodeID)
		require.Equal(t, float32(5.5), s.bottomLayerNeighbors[0].Dist)

		// Get Node 0 at Layer 1 (Upper Layer)
		n0L1, err := s.get(0, 1)
		require.NoError(t, err)
		require.Len(t, n0L1.neighbors, 16)

		// Mutate and verify it hits the upper flat array at correct offset
		n0L1.neighbors[0] = graph.NodeDistance{NodeID: 888, Dist: 4.4}
		require.Equal(t, uint32(888), s.upperLayerNeighbors[0].NodeID)

		// Get Node 0 at Layer 2 (Upper Layer)
		n0L2, err := s.get(0, 2)
		require.NoError(t, err)
		require.Len(t, n0L2.neighbors, 16)

		// Mutate and verify (should be offset by M=16 because it's Layer 2)
		n0L2.neighbors[0] = graph.NodeDistance{NodeID: 777, Dist: 3.3}
		require.Equal(t, uint32(777), s.upperLayerNeighbors[16].NodeID)

		// Get Node 1 at Layer 1
		n1L1, err := s.get(1, 1)
		require.NoError(t, err)

		// Mutate and verify (Node 1 starts at offset 32 in upper array)
		n1L1.neighbors[0] = graph.NodeDistance{NodeID: 666, Dist: 2.2}
		require.Equal(t, uint32(666), s.upperLayerNeighbors[32].NodeID)
	})

	t.Run("Get node bounds and validation errors", func(t *testing.T) {
		s := createStore()
		s.add(100, 1) // Node 0

		// Out of bounds node ID
		_, err := s.get(1, 0)
		require.ErrorIs(t, err, graph.ErrInvalidID)

		// Try to access layer higher than node's highest level
		_, err = s.get(0, 2)
		require.Error(t, err)
		require.ErrorContains(t, err, "does not exist in layer")
	})

	t.Run("LayerNodeStore wrapper implements graph interface correctly", func(t *testing.T) {
		s := createStore()
		s.add(100, 2) // Node 0
		s.add(200, 0) // Node 1

		// Wrap layer 0
		lns0, err := s.getLayerNodeStore(0)
		require.NoError(t, err)
		require.Equal(t, 2, lns0.Len()) // Both nodes exist on Layer 0

		// Wrap layer 2
		lns2, err := s.getLayerNodeStore(2)
		require.NoError(t, err)
		require.Equal(t, 1, lns2.Len()) // Only Node 0 exists on Layer 2

		// Fetch via interface method
		n, err := lns2.Get(0)
		require.NoError(t, err)
		require.Equal(t, uint32(0), n.NodeID)
		require.Equal(t, uint32(100), n.VectorID)
		require.Len(t, n.Neighbors, 16)

		// Try invalid layer wrapper
		_, err = s.getLayerNodeStore(5)
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid layer")
	})
}
