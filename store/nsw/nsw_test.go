package nsw

import (
	"fmt"
	"sort"
	"testing"

	"github.com/FluffyKebab/vektor"
	"github.com/FluffyKebab/vektor/store/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeVec(id uint32, coords ...float32) vektor.InMemoryVector {
	return vektor.NewInMemoryVector(id, coords)
}

func TestNSW_FindClosest(t *testing.T) {
	// 1. Setup Vector Store
	// Assuming vector dimension is 1 for simplicity of this test
	// 1. Setup Vector Store
	// Assuming vector dimension is 1 for simplicity of this test
	vectors := vektor.NewInMemoryVectorStore(1)

	vIdA, err := vectors.Store([]float32{10})
	require.NoError(t, err)
	vIdB, err := vectors.Store([]float32{20})
	require.NoError(t, err)
	vIdC, err := vectors.Store([]float32{30})
	require.NoError(t, err)
	vIdD, err := vectors.Store([]float32{40})
	require.NoError(t, err)

	// 2. Setup NSW and Node Store
	// maxNeighbors=2, initNeighbors=2, efSearch=2
	nsw := New(vectors, vektor.Euclidean, 2, 2, 2, 2)

	// Create nodes in the flat nodeStore
	nodeA := nsw.nodes.newNode(vIdA)
	nodeB := nsw.nodes.newNode(vIdB)
	nodeC := nsw.nodes.newNode(vIdC)
	nodeD := nsw.nodes.newNode(vIdD)

	// Set entry node
	nsw.entryNodeID = nodeA.nodeID

	// Setup a linear graph: NodeA(10) <-> NodeB(20) <-> NodeC(30) <-> NodeD(40)
	// Using nsw.AddEdge guarantees we use fresh slice views from the nodeStore
	require.NoError(t, nsw.AddEdge(nodeA.nodeID, nodeB.nodeID, 10.0))

	require.NoError(t, nsw.AddEdge(nodeB.nodeID, nodeA.nodeID, 10.0))
	require.NoError(t, nsw.AddEdge(nodeB.nodeID, nodeC.nodeID, 10.0))

	require.NoError(t, nsw.AddEdge(nodeC.nodeID, nodeB.nodeID, 10.0))
	require.NoError(t, nsw.AddEdge(nodeC.nodeID, nodeD.nodeID, 10.0))

	require.NoError(t, nsw.AddEdge(nodeD.nodeID, nodeC.nodeID, 10.0))

	fmt.Println(nsw.nodes.neighbors)

	t.Run("Finds exact match when target exists", func(t *testing.T) {
		target := vektor.NewInMemoryVector(99, []float32{40}) // Same value as NodeD

		results, err := nsw.FindClosest(target, 1)
		require.NoError(t, err)
		require.Len(t, results, 1, "Should return exactly k elements")

		// result vector ID should match nodeD's store ID
		require.Equal(t, nodeD.nodeID, results[0].ID())
	})

	t.Run("Finds k nearest neighbors correctly ordered", func(t *testing.T) {
		target := vektor.NewInMemoryVector(99, []float32{32})
		// Distances: C(30)=2, D(40)=8, B(20)=12

		// Looking for top 3 nearest neighbors
		results, err := nsw.FindClosest(target, 3)
		require.NoError(t, err)
		require.Len(t, results, 3)

		// results slice should contain C, D, B in that exact order
		expectedIDs := []uint32{nodeC.nodeID, nodeD.nodeID, nodeB.nodeID}

		var actualIDs []uint32
		for _, r := range results {
			actualIDs = append(actualIDs, r.ID())
		}

		require.Equal(t, expectedIDs, actualIDs, "Should find the 3 closest nodes in correct order")
	})

	t.Run("Distance cache (visited set) prevents redundant calculations", func(t *testing.T) {
		target := vektor.NewInMemoryVector(99, []float32{25}) // Between B and C

		callCount := 0
		originalDist := nsw.dist

		// Intercept the distance function to count calls
		nsw.dist = func(a, b []float32) (float32, error) {
			callCount++
			return originalDist(a, b)
		}
		defer func() { nsw.dist = originalDist }()

		_, err := nsw.FindClosest(target, 2)
		require.NoError(t, err)

		// Because we check the visited set `s.insertedInQueue` *before* calculating
		// distances, we should not recalculate the same node twice.
		require.LessOrEqual(t, callCount, 4, "Distance should be calculated at most once per traversed node")
	})

	t.Run("Finds closest when there is only the entry node", func(t *testing.T) {
		// Create a completely fresh NSW and Vector store to isolate state
		soloStore := vektor.NewInMemoryVectorStore(1)
		soloNSW := New(soloStore, vektor.Euclidean, 16, 16, 200, 200)

		// Manually inject the entry node
		vIdSolo, err := soloStore.Store([]float32{50})
		require.NoError(t, err)
		nodeSolo := soloNSW.nodes.newNode(vIdSolo)
		soloNSW.entryNodeID = nodeSolo.nodeID

		target := vektor.NewInMemoryVector(99, []float32{42})

		results, err := soloNSW.FindClosest(target, 3)

		require.NoError(t, err)
		require.Len(t, results, 1, "Should gracefully return only 1 result if graph only has 1 node")
		require.Equal(t, nodeSolo.nodeID, results[0].ID())
	})
}

func TestNSW_Insert(t *testing.T) {
	t.Run("Sets entry node on first insert", func(t *testing.T) {
		nsw := New(vektor.NewInMemoryVectorStore(2), vektor.Euclidean, 3, 3, 3, 3)

		vec1 := makeVec(1, 10.0, 10.0)
		_, err := nsw.Insert(vec1)

		require.NoError(t, err)
		require.NotEqual(t, graph.EmptyNodeID, nsw.entryNodeID)
		require.Equal(t, 0, int(nsw.entryNodeID))

		entryNode, _ := nsw.nodes.get(nsw.entryNodeID)
		require.Equal(t, 0, int(entryNode.vectorID))
		require.Zero(t, entryNode.neighbors.Len())
	})

	t.Run("Creates bi-directional edges on second insert", func(t *testing.T) {
		nsw := New(vektor.NewInMemoryVectorStore(2), vektor.Euclidean, 3, 3, 3, 3)

		vec1 := makeVec(0, 10.0, 10.0)
		vec2 := makeVec(0, 20.0, 20.0)

		_, err := nsw.Insert(vec1)
		require.NoError(t, err)
		_, err = nsw.Insert(vec2)
		require.NoError(t, err)

		entryNode, _ := nsw.nodes.get(nsw.entryNodeID)
		require.Equal(t, 1, entryNode.neighbors.Len())
		require.Equal(t, 1, int(entryNode.neighbors[0].NodeID))

		vec2Node, ok := nsw.nodes.get(entryNode.neighbors[0].NodeID)
		require.True(t, ok)
		require.Equal(t, 1, vec2Node.neighbors.Len())
		require.Equal(t, 0, int(vec2Node.neighbors[0].NodeID))
	})

	t.Run("Respects maxNeighbors and prunes worst edge", func(t *testing.T) {
		nsw := New(vektor.NewInMemoryVectorStore(1), vektor.Euclidean, 2, 2, 2, 2)

		// Insert 4 points sequentially on a 1D line: 10, 20, 30, 40
		// Because maxNeighbors is 2, the nodes in the middle must drop their longest
		// connections to accept closer ones.
		_, err := nsw.Insert(makeVec(0, 10))
		require.NoError(t, err)
		_, err = nsw.Insert(makeVec(0, 20))
		require.NoError(t, err)
		_, err = nsw.Insert(makeVec(0, 30))
		require.NoError(t, err)
		_, err = nsw.Insert(makeVec(0, 40))
		require.NoError(t, err)

		// Do a quick BFS to grab all nodes and check their neighbor counts
		visited := make(map[uint32]bool)
		entryNode, _ := nsw.nodes.get(nsw.entryNodeID)
		queue := []node{entryNode}

		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]

			if visited[curr.nodeID] {
				continue
			}
			visited[curr.nodeID] = true

			// Assert no node exceeded maxNeighbors
			assert.LessOrEqual(t, curr.neighbors.Len(), nsw.maxNeighbors, "Node %d exceeded max neighbors", curr.nodeID)

			for _, neighbor := range curr.neighbors {
				if neighbor.NodeID == graph.EmptyNodeID {
					break
				}

				n, ok := nsw.nodes.get(neighbor.NodeID)
				require.True(t, ok)
				queue = append(queue, n)
			}
		}

		require.Len(t, visited, 4, "Graph should contain all 4 nodes")
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

func TestNSW_InsertAndFindClosest(t *testing.T) {
	vStore := vektor.NewInMemoryVectorStore(2)
	nsw := New(vStore, vektor.Euclidean, 10, 5, 10, 10)

	var dataset []vektor.Vector

	// 1. Build a graph of 100 vectors in a 2D spatial grid (10x10)
	var id uint32
	for x := float32(0.0); x < 100.0; x += 10.0 {
		for y := float32(0.0); y < 100.0; y += 10.0 {
			vec := makeVec(id, x, y)
			dataset = append(dataset, vec)

			_, err := nsw.Insert(vec)
			require.NoError(t, err, "Insert failed at ID %d", id)

			id++
		}
	}

	// 2. Define a target
	target := makeVec(999, 46.0, 46.0)
	k := 5

	// 3. Find the mathematical ground truth
	expectedResults := bruteForceKNN(target, dataset, k)

	// 4. Query the NSW Graph
	actualResults, err := nsw.FindClosest(target, k)
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

	// Using ElementsMatch because approximate search might return exact top K
	// but in a slightly different internal order depending on priority queue state
	assert.ElementsMatch(t, expectedIDs, actualIDs,
		"NSW Search did not match brute-force k-NN ground truth")
}

func TestNSW_AddNode(t *testing.T) {
	t.Run("Successfully adds a new node", func(t *testing.T) {
		nsw := &NSW{
			nodes:        newNodeStore(16),
			maxNeighbors: 16,
		}

		err := nsw.AddNode(100) // 100 is the vectorStoreID
		require.NoError(t, err)

		// Verify it was added to the nodeStore at internal ID 0
		addedNode, exists := nsw.nodes.get(0)
		require.True(t, exists)
		require.NotNil(t, addedNode)
		assert.Equal(t, uint32(100), addedNode.vectorID)
		assert.Equal(t, 1, nsw.nodes.numNodes())
	})
}

func TestNSW_AddEdge(t *testing.T) {
	t.Run("Successfully adds an edge with distance", func(t *testing.T) {
		nsw := &NSW{
			nodes: newNodeStore(5),
		}

		n1 := nsw.nodes.newNode(100)
		n2 := nsw.nodes.newNode(200)

		err := nsw.AddEdge(n1.nodeID, n2.nodeID, 5.42)
		require.NoError(t, err)

		n1, ok := nsw.nodes.get(n1.nodeID)
		require.True(t, ok)
		n2, ok = nsw.nodes.get(n2.nodeID)
		require.True(t, ok)

		// Verify neighbor was appended correctly to n1
		require.Equal(t, 1, n1.neighbors.Len())
		assert.Equal(t, n2.nodeID, n1.neighbors[0].NodeID)
		assert.Equal(t, float32(5.42), n1.neighbors[0].Dist)

		// Verify n2 was NOT modified (directed edge behavior)
		assert.Equal(t, 0, n2.neighbors.Len())
	})

	t.Run("Errors when fromID is missing", func(t *testing.T) {
		nsw := &NSW{nodes: newNodeStore(5)}
		nsw.nodes.newNode(100) // Creates node 0 (toID)

		// Attempt to add edge from non-existent node 1
		err := nsw.AddEdge(1, 0, 5.42)
		require.Error(t, err)
		assert.ErrorContains(t, err, "from_node 1 does not exist")
	})

	t.Run("Errors when toID is missing", func(t *testing.T) {
		nsw := &NSW{nodes: newNodeStore(5)}
		nsw.nodes.newNode(100) // Creates node 0 (fromID)

		// Attempt to add edge to non-existent node 1
		err := nsw.AddEdge(0, 1, 5.42)
		require.Error(t, err)
		assert.ErrorContains(t, err, "to_node 1 does not exist")
	})
}

func TestNSW_RemoveEdge(t *testing.T) {
	t.Run("Successfully removes an existing edge", func(t *testing.T) {
		nsw := &NSW{nodes: newNodeStore(5)}

		n1 := nsw.nodes.newNode(100) // nodeID 0
		n2 := nsw.nodes.newNode(200) // nodeID 1

		// Link them
		nsw.AddEdge(n1.nodeID, n2.nodeID, 3.14)

		n1, _ = nsw.nodes.get(0) // nodeID 0
		require.Equal(t, 1, n1.neighbors.Len())

		// Remove the edge
		err := nsw.RemoveEdge(n1.nodeID, n2.nodeID)
		require.NoError(t, err)

		// In the new array implementation, removing an edge replaces it with graph.EmptyNodeID
		assert.Equal(t, 0, n1.neighbors.Len())
		assert.Equal(t, graph.EmptyNodeID, n1.neighbors[0].NodeID)
	})

	t.Run("Errors when fromID is missing", func(t *testing.T) {
		nsw := &NSW{nodes: newNodeStore(5)}
		nsw.nodes.newNode(100) // Creates node 0

		err := nsw.RemoveEdge(1, 0)
		require.Error(t, err)
		assert.ErrorContains(t, err, "from_node 1 does not exist")
	})

	t.Run("Errors when toID is missing", func(t *testing.T) {
		nsw := &NSW{nodes: newNodeStore(5)}
		nsw.nodes.newNode(100) // Creates node 0

		err := nsw.RemoveEdge(0, 1)
		require.Error(t, err)
		assert.ErrorContains(t, err, "to_node 1 does not exist")
	})

	t.Run("Errors when edge does not exist", func(t *testing.T) {
		nsw := &NSW{nodes: newNodeStore(5)}

		n1 := nsw.nodes.newNode(100) // nodeID 0
		n2 := nsw.nodes.newNode(200) // nodeID 1

		// Nodes exist, but n1 has no neighbors
		err := nsw.RemoveEdge(n1.nodeID, n2.nodeID)
		require.Error(t, err)
		assert.ErrorContains(t, err, "does not have neighbor")
	})
}
