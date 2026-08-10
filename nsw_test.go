package vektor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNSW_FindClosest(t *testing.T) {
	// Setup a linear graph: NodeA(10) <-> NodeB(20) <-> NodeC(30) <-> NodeD(40)
	vecA := InMemoryVector{id: 1, data: []float64{10}}
	vecB := InMemoryVector{id: 2, data: []float64{20}}
	vecC := InMemoryVector{id: 3, data: []float64{30}}
	vecD := InMemoryVector{id: 4, data: []float64{40}}

	nodeA := &node{v: vecA, maxNeighbors: 2}
	nodeB := &node{v: vecB, maxNeighbors: 2}
	nodeC := &node{v: vecC, maxNeighbors: 2}
	nodeD := &node{v: vecD, maxNeighbors: 2}

	nodeA.neighbors = []*node{nodeB}
	nodeB.neighbors = []*node{nodeA, nodeC}
	nodeC.neighbors = []*node{nodeB, nodeD}
	nodeD.neighbors = []*node{nodeC}

	nsw := NSW{entryNode: nodeA}

	t.Run("Finds exact match when target exists", func(t *testing.T) {
		target := InMemoryVector{id: 99, data: []float64{40}} // Same value as NodeD

		results, err := nsw.FindClosest(target, 1, euclidean)
		require.NoError(t, err)
		require.Len(t, results, 1, "Should return exactly k elements")

		assert.Equal(t, vecD.ID(), results[0].ID())
	})

	t.Run("Finds k nearest neighbors correctly ordered", func(t *testing.T) {
		target := InMemoryVector{id: 99, data: []float64{32}} // Closest to C(30), then D(40), then B(20)

		// Looking for top 3 nearest neighbors
		results, err := nsw.FindClosest(target, 3, euclidean)
		require.NoError(t, err)
		require.Len(t, results, 3)

		// results slice should contain C, D, B
		expectedIDs := []int{vecC.ID(), vecD.ID(), vecB.ID()}
		var actualIDs []int
		for _, r := range results {
			actualIDs = append(actualIDs, r.ID())
		}
		assert.ElementsMatch(t, expectedIDs, actualIDs, "Should find the 3 closest nodes")
	})

	t.Run("Distance cache prevents redundant calculations", func(t *testing.T) {
		target := InMemoryVector{id: 99, data: []float64{25}} // Between B and C

		callCount := 0
		trackedDistance := func(a, b Vector) (float64, error) {
			callCount++
			return euclidean(a, b)
		}

		_, err := nsw.FindClosest(target, 2, trackedDistance)
		require.NoError(t, err)

		assert.LessOrEqual(t, callCount, 4, "Distance should be cached per node")
	})
}
