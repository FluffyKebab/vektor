package hnsw

import (
	"github.com/FluffyKebab/vektor"
	"github.com/FluffyKebab/vektor/store/graph"
)

type HNSW struct {
	nodes   *nodeStore
	vectors *vektor.Vector

	candidateQueue *graph.HeapQueue
	resultQueue    *graph.SliceQueue
	visited        *graph.VisitedSet
	dist           vektor.Distance
}
