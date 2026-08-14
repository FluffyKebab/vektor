package store_test

import (
	"context"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FluffyKebab/vektor"
	"github.com/FluffyKebab/vektor/store"
)

func TestStore_RebootIntegration(t *testing.T) {
	// 1. Setup isolated temporary files for the test
	tempDir := t.TempDir()
	graphWALPath := filepath.Join(tempDir, "graph.wal")
	vectorDataPath := filepath.Join(tempDir, "vectors.dat")

	vectorSize := 2

	// --- PHASE 1: BUILD AND SAVE ---
	ctx1, cancel1 := context.WithCancel(context.Background())

	db1, err := store.New(ctx1, vektor.Euclidean, vectorSize, graphWALPath, vectorDataPath)
	require.NoError(t, err, "Failed to initialize first DB instance")

	// Insert 200 vectors in a straight line
	for i := uint32(0); i < 200; i++ {
		vec := vektor.NewInMemoryVector(i, []float64{float64(i), float64(i * 2)})
		_, err := db1.Insert(vec)
		require.NoError(t, err, "Failed to insert vector ID %d", i)
	}

	// Define a target we want to search for
	target := vektor.NewInMemoryVector(999, []float64{50.4, 100.8})

	// Search the in-memory graph before shutdown
	results1, err := db1.FindClosest(target, 5)
	require.NoError(t, err)
	require.Len(t, results1, 5)

	// Save the ID of the absolute closest node to verify later
	expectedClosestID := results1[0].ID()

	// Shut down DB1 gracefully
	cancel1()

	// Wait a moment for the goroutine running s.handleClose(ctx) to close the file handles,
	// otherwise Phase 2 might fail with a "file in use" error on Windows.
	time.Sleep(100 * time.Millisecond)

	// --- PHASE 2: REBOOT AND VERIFY ---
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2() // Cleanup at end of test

	// Boot up a completely new instance pointing to the same files
	db2, err := store.New(ctx2, vektor.Euclidean, vectorSize, graphWALPath, vectorDataPath)
	require.NoError(t, err, "Failed to reboot DB from disk")

	// Search the newly restored graph for the exact same target
	results2, err := db2.FindClosest(target, 5)
	require.NoError(t, err)
	require.Len(t, results2, 5)

	// ASSERTION: The rebooted graph must route to the exact same closest node
	assert.Equal(t, expectedClosestID, results2[0].ID(),
		"Rebooted graph did not find the same closest node. WAL replay failed.")

	// --- PHASE 3: LIVE MUTATION AFTER REBOOT ---
	// Prove the database is fully functional after restoring from disk
	vec201 := vektor.NewInMemoryVector(201, []float64{201.0, 402.0})
	_, err = db2.Insert(vec201)
	require.NoError(t, err, "Failed to insert new vector after reboot")

	results3, err := db2.FindClosest(vec201, 1)
	require.NoError(t, err)
	assert.Equal(t, uint32(201), results3[0].ID(), "Graph failed to route to newly inserted node")
}

// generateRandomVector creates a vector with values between 0.0 and 1000.0
func generateRandomVector(id uint32, dimensions int, r *rand.Rand) vektor.Vector {
	coords := make([]float64, dimensions)
	for i := 0; i < dimensions; i++ {
		coords[i] = r.Float64() * 1000.0
	}
	return vektor.NewInMemoryVector(id, coords)
}

func TestStore_ExtensiveRebootIntegration(t *testing.T) {
	// 1. Setup isolated temporary files for the test
	tempDir := t.TempDir()
	graphWALPath := filepath.Join(tempDir, "graph.wal")
	vectorDataPath := filepath.Join(tempDir, "vectors.dat")

	vectorSize := 128 // High dimensionality to stress test the routing

	// Use a fixed seed for deterministic testing. If the test fails,
	// it will fail exactly the same way every time you run it.
	r := rand.New(rand.NewSource(42))

	// --- PHASE 1: BUILD, SEARCH, AND SAVE ---
	ctx1, cancel1 := context.WithCancel(context.Background())

	db1, err := store.New(ctx1, vektor.Euclidean, vectorSize, graphWALPath, vectorDataPath)
	require.NoError(t, err, "Failed to initialize first DB instance")

	// Insert 500 random vectors to build a complex graph topology
	for i := uint32(1); i <= 500; i++ {
		vec := generateRandomVector(i, vectorSize, r)
		_, err := db1.Insert(vec)
		require.NoError(t, err, "Failed to insert vector ID %d", i)
	}

	// Generate 100 random search queries and record their results
	k := 5
	queries := make([]vektor.Vector, 100)
	preRebootResults := make([][]uint32, 100)

	for i := 0; i < 100; i++ {
		// Use high IDs for queries so they don't collide with stored data (1-500)
		queryVec := generateRandomVector(uint32(9999+i), vectorSize, r)
		queries[i] = queryVec

		results, err := db1.FindClosest(queryVec, k)
		require.NoError(t, err, "Pre-reboot search %d failed", i)
		require.Len(t, results, k, "Pre-reboot search %d did not return k results", i)

		// Store just the IDs of the top K results to compare later
		var resultIDs []uint32
		for _, v := range results {
			resultIDs = append(resultIDs, v.ID())
		}
		preRebootResults[i] = resultIDs
	}

	// Shut down DB1 gracefully to flush everything to disk
	cancel1()

	// Ensure file handles are fully released by the OS
	time.Sleep(200 * time.Millisecond)

	// --- PHASE 2: REBOOT AND VERIFY ---
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	db2, err := store.New(ctx2, vektor.Euclidean, vectorSize, graphWALPath, vectorDataPath)
	require.NoError(t, err, "Failed to reboot DB from disk")

	// Run the exact same 100 queries against the rebooted graph
	for i := 0; i < 100; i++ {
		queryVec := queries[i]

		results, err := db2.FindClosest(queryVec, k)
		require.NoError(t, err, "Post-reboot search %d failed", i)
		require.Len(t, results, k, "Post-reboot search %d did not return k results", i)

		var postRebootIDs []uint32
		for _, v := range results {
			postRebootIDs = append(postRebootIDs, v.ID())
		}

		// ASSERTION: The graph topology must be exactly identical.
		// If it is, the greedy search will traverse the exact same path
		// and return the exact same Top K nodes in the exact same order.
		require.Equal(t, preRebootResults[i], postRebootIDs,
			"Search mismatch on query %d (Seed ID: %d). WAL replay failed to perfectly reconstruct graph.", i, queryVec.ID())
	}
}
