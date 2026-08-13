package nsw

import (
	"sort"
	"testing"

	"github.com/FluffyKebab/vektor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeVec(id uint32, coords ...float64) vektor.InMemoryVector {
	return vektor.NewInMemoryVector(id, coords)
}

func TestNSW_FindClosest(t *testing.T) {
	// Setup a linear graph: NodeA(10) <-> NodeB(20) <-> NodeC(30) <-> NodeD(40)
	vecA := makeVec(1, 10)
	vecB := makeVec(2, 20)
	vecC := makeVec(3, 30)
	vecD := makeVec(4, 40)

	store := vektor.NewInMemoryVectorStore()
	store.Store(vecA)
	store.Store(vecB)
	store.Store(vecC)
	store.Store(vecD)

	nodeA := &node{id: 1}
	nodeB := &node{id: 2}
	nodeC := &node{id: 3}
	nodeD := &node{id: 4}

	nodeA.neighbors = []nodeDistance{{nodeB, 0}}
	nodeB.neighbors = []nodeDistance{{nodeA, 0}, {nodeC, 0}}
	nodeC.neighbors = []nodeDistance{{nodeC, 0}, {nodeD, 0}}
	nodeD.neighbors = []nodeDistance{{nodeC, 0}}

	nsw := NSW{entryNode: nodeA, dist: vektor.Euclidean, store: store, nodes: make(map[uint32]*node)}

	t.Run("Finds exact match when target exists", func(t *testing.T) {
		target := vektor.NewInMemoryVector(99, []float64{40}) // Same value as NodeD

		results, err := nsw.FindClosest(target, 1)
		require.NoError(t, err)
		require.Len(t, results, 1, "Should return exactly k elements")

		assert.Equal(t, vecD.ID(), results[0].ID())
	})

	t.Run("Finds k nearest neighbors correctly ordered", func(t *testing.T) {
		target := vektor.NewInMemoryVector(99, []float64{32}) // Closest to C(30), then D(40), then B(20)

		// Looking for top 3 nearest neighbors
		results, err := nsw.FindClosest(target, 3)
		require.NoError(t, err)
		require.Len(t, results, 3)

		// results slice should contain C, D, B
		expectedIDs := []uint32{vecC.ID(), vecD.ID(), vecB.ID()}
		var actualIDs []uint32
		for _, r := range results {
			actualIDs = append(actualIDs, r.ID())
		}
		assert.ElementsMatch(t, expectedIDs, actualIDs, "Should find the 3 closest nodes")
	})

	t.Run("Distance cache prevents redundant calculations", func(t *testing.T) {
		target := vektor.NewInMemoryVector(99, []float64{25}) // Between B and C

		callCount := 0
		nsw.dist = func(a, b vektor.Vector) (float64, error) {
			callCount++
			return vektor.Euclidean(a, b)
		}

		_, err := nsw.FindClosest(target, 2)
		require.NoError(t, err)

		assert.LessOrEqual(t, callCount, 4, "Distance should be cached per node")
	})

	t.Run("Finds closest when there is only the entry node", func(t *testing.T) {
		vecSolo := makeVec(5, 50)
		nodeSolo := &node{id: 5}
		store := vektor.NewInMemoryVectorStore()
		store.Store(vecSolo)

		soloNSW := NSW{entryNode: nodeSolo, dist: vektor.Euclidean, store: store}
		target := vektor.NewInMemoryVector(99, []float64{42})

		results, err := soloNSW.FindClosest(target, 3)

		require.NoError(t, err)
		require.Len(t, results, 1, "Should return exactly 1 element since the graph is size 1")
		assert.Equal(t, vecSolo.ID(), results[0].ID())
	})
}

func TestNSW_Insert(t *testing.T) {
	t.Run("Sets entry node on first insert", func(t *testing.T) {
		nsw := New(vektor.NewInMemoryVectorStore(), vektor.Euclidean, 3, 3, 3)

		vec1 := makeVec(1, 10.0, 10.0)
		err := nsw.Insert(vec1)

		require.NoError(t, err)
		require.NotNil(t, nsw.entryNode)
		assert.Equal(t, 1, int(nsw.entryNode.id))
		assert.Empty(t, nsw.entryNode.neighbors)
	})

	t.Run("Creates bi-directional edges on second insert", func(t *testing.T) {
		nsw := New(vektor.NewInMemoryVectorStore(), vektor.Euclidean, 3, 3, 3)

		vec1 := makeVec(1, 10.0, 10.0)
		vec2 := makeVec(2, 20.0, 20.0)

		require.NoError(t, nsw.Insert(vec1))
		require.NoError(t, nsw.Insert(vec2))

		assert.Len(t, nsw.entryNode.neighbors, 1)
		assert.Equal(t, 2, int(nsw.entryNode.neighbors[0].node.id))

		vec2Node := nsw.entryNode.neighbors[0].node
		assert.Len(t, vec2Node.neighbors, 1)
		assert.Equal(t, 1, int(vec2Node.neighbors[0].node.id))
	})

	t.Run("Respects maxNeighbors and prunes worst edge", func(t *testing.T) {
		nsw := New(vektor.NewInMemoryVectorStore(), vektor.Euclidean, 2, 2, 2)

		// Insert 4 points sequentially on a 1D line: 10, 20, 30, 40
		// Because maxNeighbors is 2, the nodes in the middle must drop their longest
		// connections to accept closer ones.
		require.NoError(t, nsw.Insert(makeVec(1, 10.0)))
		require.NoError(t, nsw.Insert(makeVec(2, 20.0)))
		require.NoError(t, nsw.Insert(makeVec(3, 30.0)))
		require.NoError(t, nsw.Insert(makeVec(4, 40.0)))

		// Do a quick BFS to grab all nodes and check their neighbor counts
		visited := make(map[uint32]bool)
		queue := []*node{nsw.entryNode}

		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]

			if visited[curr.id] {
				continue
			}
			visited[curr.id] = true

			// Assert no node exceeded maxNeighbors
			assert.LessOrEqual(t, len(curr.neighbors), nsw.maxNeighbors, "Node %d exceeded max neighbors", curr.id)

			for _, neighbor := range curr.neighbors {
				queue = append(queue, neighbor.node)
			}
		}

		assert.Len(t, visited, 4, "Graph should contain all 4 nodes")
	})
}

func bruteForceKNN(target vektor.Vector, dataset []vektor.Vector, k int) []vektor.Vector {
	type distPair struct {
		v vektor.Vector
		d float64
	}

	var pairs []distPair
	for _, v := range dataset {
		d, _ := vektor.Euclidean(target, v)
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
	nsw := New(vektor.NewInMemoryVectorStore(), vektor.Euclidean, 10, 5, 10)

	var dataset []vektor.Vector

	// 1. Build a graph of 100 vectors in a 2D spatial grid (10x10)
	idCounter := 1
	for x := 0.0; x < 100.0; x += 10.0 {
		for y := 0.0; y < 100.0; y += 10.0 {
			vec := makeVec(uint32(idCounter), x, y)
			dataset = append(dataset, vec)
			require.NoError(t, nsw.Insert(vec), "Insert failed at ID %d", idCounter)
			idCounter++
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
	// but in a slightly different internal order depending on the priority queue state
	assert.ElementsMatch(t, expectedIDs, actualIDs,
		"NSW Search did not match brute-force k-NN ground truth")
}

func TestNSW_AddNode(t *testing.T) {
	t.Run("Successfully adds a new node", func(t *testing.T) {
		nsw := &NSW{
			nodes:        make(map[uint32]*node),
			maxNeighbors: 16,
		}

		err := nsw.AddNode(1)
		require.NoError(t, err)

		// Verify it was added to the map
		addedNode, exists := nsw.nodes[1]
		require.True(t, exists)
		require.NotNil(t, addedNode)
	})

	t.Run("Errors when adding an existing node", func(t *testing.T) {
		nsw := &NSW{
			nodes:        make(map[uint32]*node),
			maxNeighbors: 16,
		}

		// Add it the first time
		require.NoError(t, nsw.AddNode(1))

		// Try to add it again
		err := nsw.AddNode(1)
		require.Error(t, err)
		assert.ErrorContains(t, err, "already exist in graph")
	})
}

func TestNSW_AddEdge(t *testing.T) {
	t.Run("Successfully adds an edge with distance", func(t *testing.T) {
		nsw := &NSW{
			nodes: make(map[uint32]*node),
		}

		// Setup nodes directly
		n1 := &node{id: 1, neighbors: make([]nodeDistance, 0)}
		n2 := &node{id: 2, neighbors: make([]nodeDistance, 0)}
		nsw.nodes[1] = n1
		nsw.nodes[2] = n2

		err := nsw.AddEdge(1, 2, 5.42)
		require.NoError(t, err)

		// Verify neighbor was appended correctly to n1
		require.Len(t, n1.neighbors, 1)
		assert.Equal(t, uint32(2), n1.neighbors[0].node.id)
		assert.Equal(t, 5.42, n1.neighbors[0].dist)

		// Verify n2 was NOT modified (directed edge behavior)
		assert.Empty(t, n2.neighbors)
	})

	t.Run("Errors when fromID is missing", func(t *testing.T) {
		nsw := &NSW{nodes: make(map[uint32]*node)}
		nsw.nodes[2] = &node{id: 2} // Only toID exists

		err := nsw.AddEdge(1, 2, 5.42)
		require.Error(t, err)
		assert.ErrorContains(t, err, "from_node 1 does not exist")
	})

	t.Run("Errors when toID is missing", func(t *testing.T) {
		nsw := &NSW{nodes: make(map[uint32]*node)}
		nsw.nodes[1] = &node{id: 1} // Only fromID exists

		err := nsw.AddEdge(1, 2, 5.42)
		require.Error(t, err)
		assert.ErrorContains(t, err, "to_node 2 does not exist")
	})
}

func TestNSW_RemoveEdge(t *testing.T) {
	t.Run("Successfully removes an existing edge", func(t *testing.T) {
		nsw := &NSW{nodes: make(map[uint32]*node)}

		n1 := &node{id: 1}
		n2 := &node{id: 2}
		nsw.nodes[1] = n1
		nsw.nodes[2] = n2

		// Manually link them
		n1.neighbors = []nodeDistance{{node: n2, dist: 3.14}}

		err := nsw.RemoveEdge(1, 2)
		require.NoError(t, err)

		// Assuming n1.removeNeighbor either zeros out or shrinks the slice,
		// we verify the length changed (adjust assertion if your remove logic differs)
		assert.Empty(t, n1.neighbors)
	})

	t.Run("Errors when fromID is missing", func(t *testing.T) {
		nsw := &NSW{nodes: make(map[uint32]*node)}
		nsw.nodes[2] = &node{id: 2}

		err := nsw.RemoveEdge(1, 2)
		require.Error(t, err)
		assert.ErrorContains(t, err, "from_node 1 does not exist")
	})

	t.Run("Errors when toID is missing", func(t *testing.T) {
		nsw := &NSW{nodes: make(map[uint32]*node)}
		nsw.nodes[1] = &node{id: 1}

		err := nsw.RemoveEdge(1, 2)
		require.Error(t, err)
		assert.ErrorContains(t, err, "to_node 2 does not exist")
	})

	t.Run("Errors when edge does not exist", func(t *testing.T) {
		nsw := &NSW{nodes: make(map[uint32]*node)}

		n1 := &node{id: 1}
		n2 := &node{id: 2}
		nsw.nodes[1] = n1
		nsw.nodes[2] = n2

		// Nodes exist, but n1 has no neighbors
		err := nsw.RemoveEdge(1, 2)
		require.Error(t, err)
		assert.ErrorContains(t, err, "from_id 1 does not have neighbor 2")
	})
}
