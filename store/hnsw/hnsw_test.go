package hnsw

import (
	"sort"
	"testing"

	"github.com/FluffyKebab/vektor"
	"github.com/FluffyKebab/vektor/store/graph"
	"github.com/stretchr/testify/require"
)

func makeVec(id uint32, coords ...float32) vektor.InMemoryVector {
	return vektor.NewInMemoryVector(id, coords)
}

func newTestHNSW(store vektor.VectorStore, layers, maxN0, maxN1 int) *HNSW {
	return &HNSW{
		nodes:             newNodeStore(layers, maxN0, maxN1, 100),
		vectors:           store,
		dist:              vektor.Euclidean,
		entryNodeID:       graph.EmptyNodeID,
		maxIterations:     400,
		efUp:              10,
		efDown:            10,
		efConstruction:    10,
		initNeighborsUp:   maxN1,
		initNeighborsDown: maxN0,
	}
}

func TestHNSW_FindClosest(t *testing.T) {
	// 1. Setup Vector Store
	vectors := vektor.NewInMemoryVectorStore(1)

	vIdA, err := vectors.Store([]float32{10})
	require.NoError(t, err)
	vIdB, err := vectors.Store([]float32{20})
	require.NoError(t, err)
	vIdC, err := vectors.Store([]float32{30})
	require.NoError(t, err)
	vIdD, err := vectors.Store([]float32{40})
	require.NoError(t, err)

	// 2. Setup a 1-Layer HNSW (Acts like a flat graph for testing base mechanics)
	hnsw := newTestHNSW(vectors, 1, 2, 2)

	// Create nodes in the nodeStore on Layer 0
	nodeIdA, err := hnsw.nodes.add(vIdA, 0)
	require.NoError(t, err)
	nodeIdB, err := hnsw.nodes.add(vIdB, 0)
	require.NoError(t, err)
	nodeIdC, err := hnsw.nodes.add(vIdC, 0)
	require.NoError(t, err)
	nodeIdD, err := hnsw.nodes.add(vIdD, 0)
	require.NoError(t, err)

	hnsw.entryNodeID = nodeIdA

	// Fetch nodes to add neighbors
	nodeA, _ := hnsw.nodes.get(nodeIdA, 0)
	nodeB, _ := hnsw.nodes.get(nodeIdB, 0)
	nodeC, _ := hnsw.nodes.get(nodeIdC, 0)
	nodeD, _ := hnsw.nodes.get(nodeIdD, 0)

	// Setup a linear graph: NodeA(10) <-> NodeB(20) <-> NodeC(30) <-> NodeD(40)
	hnsw.addNeighbor(0, nodeA, nodeIdB, 10.0)
	hnsw.addNeighbor(0, nodeB, nodeIdA, 10.0)

	hnsw.addNeighbor(0, nodeB, nodeIdC, 10.0)
	hnsw.addNeighbor(0, nodeC, nodeIdB, 10.0)

	hnsw.addNeighbor(0, nodeC, nodeIdD, 10.0)
	hnsw.addNeighbor(0, nodeD, nodeIdC, 10.0)

	t.Run("Finds exact match when target exists", func(t *testing.T) {
		target := vektor.NewInMemoryVector(99, []float32{40}) // Same value as NodeD

		results, err := hnsw.FindClosest(target, 1)
		require.NoError(t, err)
		require.Len(t, results, 1, "Should return exactly k elements")

		require.Equal(t, nodeIdD, results[0].ID())
	})

	t.Run("Finds k nearest neighbors correctly ordered", func(t *testing.T) {
		target := vektor.NewInMemoryVector(99, []float32{32})
		// Distances: C(30)=2, D(40)=8, B(20)=12

		// Looking for top 3 nearest Neighbors
		results, err := hnsw.FindClosest(target, 3)
		require.NoError(t, err)
		require.Len(t, results, 3)

		expectedIDs := []uint32{nodeIdC, nodeIdD, nodeIdB}

		var actualIDs []uint32
		for _, r := range results {
			actualIDs = append(actualIDs, r.ID())
		}

		require.Equal(t, expectedIDs, actualIDs, "Should find the 3 closest nodes in correct order")
	})

	t.Run("Distance cache (visited set) prevents redundant calculations", func(t *testing.T) {
		target := vektor.NewInMemoryVector(99, []float32{25}) // Between B and C

		callCount := 0
		originalDist := hnsw.dist

		// Intercept the distance function to count calls
		hnsw.dist = func(a, b []float32) (float32, error) {
			callCount++
			return originalDist(a, b)
		}
		defer func() { hnsw.dist = originalDist }()

		_, err := hnsw.FindClosest(target, 2)
		require.NoError(t, err)

		// Visited set should block recalculating identical node distances
		require.LessOrEqual(t, callCount, 4, "Distance should be calculated at most once per traversed node")
	})

	t.Run("Finds closest when there is only the entry node", func(t *testing.T) {
		soloStore := vektor.NewInMemoryVectorStore(1)
		soloHNSW := newTestHNSW(soloStore, 1, 16, 16)

		vIdSolo, err := soloStore.Store([]float32{50})
		require.NoError(t, err)

		nodeIdSolo, err := soloHNSW.nodes.add(vIdSolo, 0)
		require.NoError(t, err)
		soloHNSW.entryNodeID = nodeIdSolo

		target := vektor.NewInMemoryVector(99, []float32{42})

		results, err := soloHNSW.FindClosest(target, 3)
		require.NoError(t, err)
		require.Len(t, results, 1, "Should gracefully return only 1 result if graph only has 1 node")
		require.Equal(t, nodeIdSolo, results[0].ID())
	})
}

func TestHNSW_Insert(t *testing.T) {
	t.Run("Sets entry node on first insert", func(t *testing.T) {
		hnsw := newTestHNSW(vektor.NewInMemoryVectorStore(2), 4, 16, 16)

		vec1 := makeVec(1, 10.0, 10.0)
		_, err := hnsw.Insert(vec1)

		require.NoError(t, err)
		require.NotEqual(t, graph.EmptyNodeID, hnsw.entryNodeID)

		entryNode, err := hnsw.nodes.get(hnsw.entryNodeID, uint8(hnsw.nodes.layers-1))
		require.NoError(t, err)
		require.Equal(t, 0, int(entryNode.vectorStoreID))
	})

	t.Run("Creates bi-directional edges on second insert", func(t *testing.T) {
		hnsw := newTestHNSW(vektor.NewInMemoryVectorStore(2), 2, 3, 3)

		vec1 := makeVec(0, 10.0, 10.0)
		vec2 := makeVec(0, 20.0, 20.0)

		_, err := hnsw.Insert(vec1)
		require.NoError(t, err)
		_, err = hnsw.Insert(vec2)
		require.NoError(t, err)

		// Check Layer 0 connections
		entryNode, err := hnsw.nodes.get(0, 0)
		require.NoError(t, err)

		hasConnection := false
		for _, n := range entryNode.neighbors {
			if n.NodeID == 1 {
				hasConnection = true
			}
		}
		require.True(t, hasConnection, "Node 0 should have an edge to Node 1")

		node2, err := hnsw.nodes.get(1, 0)
		require.NoError(t, err)

		hasReverseConnection := false
		for _, n := range node2.neighbors {
			if n.NodeID == 0 {
				hasReverseConnection = true
			}
		}
		require.True(t, hasReverseConnection, "Node 1 should have an edge back to Node 0")
	})
}

func bruteForceKNN(target vektor.Vector, dataset []vektor.Vector, k int) []vektor.Vector {
	type distPair struct {
		v vektor.Vector
		d float32
	}

	targetData, _ := target.Read()

	var pairs []distPair
	for _, v := range dataset {
		vData, _ := v.Read()
		d, _ := vektor.Euclidean(targetData, vData)
		pairs = append(pairs, distPair{v, d})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].d < pairs[j].d
	})

	var result []vektor.Vector
	for i := 0; i < k && i < len(pairs); i++ {
		result = append(result, pairs[i].v)
	}
	return result
}

func TestHNSW_InsertAndFindClosest(t *testing.T) {
	vStore := vektor.NewInMemoryVectorStore(2)
	// Create a full HNSW graph: 4 layers, M0=16, M=8
	hnsw := newTestHNSW(vStore, 4, 16, 8)

	var dataset []vektor.Vector

	// 1. Build a graph of 100 vectors in a 2D spatial grid (10x10)
	var id uint32
	for x := float32(0.0); x < 100.0; x += 10.0 {
		for y := float32(0.0); y < 100.0; y += 10.0 {
			vec := makeVec(id, x, y)
			dataset = append(dataset, vec)

			_, err := hnsw.Insert(vec)
			require.NoError(t, err, "Insert failed at ID %d", id)
			id++
		}
	}

	// 2. Define a target
	target := makeVec(999, 46.1, 46.2)
	k := 5

	// 3. Find the mathematical ground truth
	expectedResults := bruteForceKNN(target, dataset, k)

	// 4. Query the HNSW Graph
	actualResults, err := hnsw.FindClosest(target, k)
	require.NoError(t, err)
	require.Len(t, actualResults, k)

	// 5. Compare graph search vs brute-force truth
	var expectedIDs []uint32
	var actualIDs []uint32

	for _, v := range expectedResults {
		expectedIDs = append(expectedIDs, v.ID())
	}
	for _, v := range actualResults {
		actualIDs = append(actualIDs, v.ID())
	}

	require.ElementsMatch(t, expectedIDs, actualIDs,
		"HNSW Search did not match brute-force k-NN ground truth")
}
