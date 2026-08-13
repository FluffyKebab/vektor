package store_test

import (
	"context"
	"math/rand"
	"path/filepath"
	"sort"
	"testing"

	"github.com/FluffyKebab/vektor"
	"github.com/FluffyKebab/vektor/store"
)

func generateDataset(size int, dims int, r *rand.Rand) []vektor.Vector {
	dataset := make([]vektor.Vector, size)
	for i := 0; i < size; i++ {
		coords := make([]float64, dims)
		for d := 0; d < dims; d++ {
			coords[d] = r.Float64() * 100.0
		}

		dataset[i] = vektor.NewInMemoryVector(uint32(i+1), coords)
	}
	return dataset
}

// bruteForceTopK calculates the exact mathematical nearest neighbors
func bruteForceTopK(query vektor.Vector, dataset []vektor.Vector, k int) []uint32 {
	type distPair struct {
		id   uint32
		dist float64
	}

	pairs := make([]distPair, len(dataset))
	for i, v := range dataset {
		d, _ := vektor.Euclidean(query, v)
		pairs[i] = distPair{id: v.ID(), dist: d}
	}

	// Sort by distance (ascending)
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].dist < pairs[j].dist
	})

	result := make([]uint32, k)
	for i := 0; i < k; i++ {
		result[i] = pairs[i].id
	}
	return result
}

// calcRecall compares the ANN results to the exact Ground Truth
func calcRecall(actual []vektor.Vector, truth []uint32) float64 {
	truthMap := make(map[uint32]bool, len(truth))
	for _, id := range truth {
		truthMap[id] = true
	}

	matches := 0
	for _, v := range actual {
		if truthMap[v.ID()] {
			matches++
		}
	}

	return float64(matches) / float64(len(truth))
}

func BenchmarkStore_SearchRecall(b *testing.B) {
	datasetSize := 10_000 // 10k vectors is enough to show ANN benefits
	querySize := 100      // Number of queries to average out latency/recall
	dims := 128
	k := 10

	r := rand.New(rand.NewSource(42))

	b.Logf("Generating %d vectors for dataset...", datasetSize)
	dataset := generateDataset(datasetSize, dims, r)
	queries := generateDataset(querySize, dims, r)

	b.Log("Calculating brute-force ground truth...")
	groundTruths := make([][]uint32, querySize)
	for i, q := range queries {
		groundTruths[i] = bruteForceTopK(q, dataset, k)
	}

	tempDir := b.TempDir()
	graphWALPath := filepath.Join(tempDir, "bench_graph.wal")
	vectorDataPath := filepath.Join(tempDir, "bench_vectors.dat")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := store.New(ctx, vektor.Euclidean, dims, graphWALPath, vectorDataPath)
	if err != nil {
		b.Fatalf("Failed to init db: %v", err)
	}

	b.Log("Building NSW Graph...")
	for _, v := range dataset {
		if err := db.Insert(v); err != nil {
			b.Fatalf("Failed to insert: %v", err)
		}
	}

	b.Log("Starting benchmark...")

	b.Run("Search", func(b *testing.B) {
		var totalRecall float64
		for i := 0; i < b.N; i++ {
			iterationRecall := 0.0

			for qIdx, q := range queries {
				results, err := db.FindClosest(q, k)
				if err != nil {
					b.Fatalf("Search failed: %v", err)
				}

				iterationRecall += calcRecall(results, groundTruths[qIdx])
			}

			totalRecall += iterationRecall / float64(querySize)
		}

		b.StopTimer()

		finalAvgRecall := totalRecall / float64(b.N)

		b.ReportMetric(finalAvgRecall, "recall_ratio")

		totalQueries := float64(b.N * querySize)
		b.ReportMetric(totalQueries/b.Elapsed().Seconds(), "queries/sec")
	})
}
